package kvass

import (
	qr "github.com/skip2/go-qrcode"

	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/lizebang/qrcode-terminal"
	"github.com/teris-io/cli"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

func getPersistance(options map[string]string) *SqlitePersistance {
	dbpath, contains := options["db"]
	if !contains {

		defaultFilename := ".kvassdb.sqlite"
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		dbpath = path.Join(home, defaultFilename)
	}

	p, err := NewSqlitePersistance(dbpath)
	if err != nil {
		panic(err)
	}
	return p
}
func GetApp() cli.App {
	logger := log.New(os.Stderr, "", log.Llongfile|log.LstdFlags)
	ls := cli.NewCommand("ls", "list keys").
		WithAction(func(args []string, options map[string]string) int {
			p := getPersistance(options)
			defer p.Close()

			err := p.GetRemoteUpdates()
			if err != nil {
				logger.Println("Couldn't get updates from server. ", err)
			}

			keys, err := p.GetKeys()
			if err != nil {
				panic(err)
			}

			for _, k := range keys {
				fmt.Println(k)
			}

			return 0
		})
	get := cli.NewCommand("get", "get a value").
		WithArg(cli.NewArg("key", "the key to get")).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]
			p := getPersistance(options)
			defer p.Close()

			err := p.GetRemoteUpdates()
			if err != nil {
				logger.Println("Couldn't get updates from server. ", err)
			}
			val, err := p.GetEntry(key)
			if err != nil {
				panic(err)
			}

			if val == nil {
				println()
				return 0 // no entry
			}

			_, err = io.Copy(os.Stdout, bytes.NewBuffer(val.Value))
			if err != nil {
				panic(err)
			}
			return 0
		})

	set := cli.NewCommand("set", "set a value").
		WithArg(cli.NewArg("key", "the key to set")).
		WithArg(cli.NewArg("value", "the value to set (ommit for stdin)").AsOptional()).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]

			p := getPersistance(options)
			defer p.Close()

			var err error
			var val []byte

			if len(args) < 2 {
				valBytes, err := ioutil.ReadAll(os.Stdin)
				val = valBytes
				if err != nil {
					panic(err)
				}

			} else {
				val = []byte(args[1] + "\n")
			}

			err = Set(p, key, []byte(val))
			if err != nil {
				panic(err)
			}

			if err = p.Push(); err != nil {
				fmt.Println("Could not push changes to server: ", err)
				return 1
			}
			return 0
		})

	rm := cli.NewCommand("rm", "remove a key").
		WithArg(cli.NewArg("key", "the key to remove")).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]

			p := getPersistance(options)
			defer p.Close()

			err := Delete(p, key)
			if err != nil {
				panic(err)
			}

			if err = p.Push(); err != nil {
				fmt.Println("Could not push changes to server: ", err)
				return 1
			}
			return 0
		})

	serve := cli.NewCommand("serve", "start in server mode [--bind=\"ip:port\" (default: 0.0.0.0:8000)]").
		WithOption(cli.NewOption("bind", "bind address (default: \"0.0.0.0:8000\" meaning all interfaces, port 8000)")).
		WithAction(func(args []string, options map[string]string) int {
			bind, contains := options["bind"]
			if !contains {
				bind = "0.0.0.0:8000"
			}
			p := getPersistance(options)
			defer p.Close()
			RunServer(p, bind)
			return 0
		})

	config_show := cli.NewCommand("show", "print current config").
		WithAction(func(args []string, options map[string]string) int {
			p := getPersistance(options)
			remote := "(None)"
			if p.State.RemoteURL != nil {
				remote = p.State.RemoteURL.Host
			}

			fmt.Printf("Encryption Key:  \t%v\n", p.State.Key)
			fmt.Printf("ProcessID:       \t%v\n", p.State.Pid)
			fmt.Printf("Remote:          \t%v\n", remote)
			return 0
		})

	config_key := cli.NewCommand("key", "set encryption key").
		WithArg(cli.NewArg("key", "the hex-encoded enryption key")).
		WithAction(func(args []string, options map[string]string) int {
			key_hex := args[0]
			key, err := hex.DecodeString(strings.TrimSpace(key_hex))
			if err != nil {
				fmt.Println("Error, could not decode supplied key.")
				return 1
			}

			if len(key) != 32 {
				fmt.Println("Error, key has to be 32 bytes long.")
				return 1
			}

			p := getPersistance(options)
			p.State.Key = key_hex
			err = p.CommitState()
			if err != nil {
				fmt.Println("Internal error: ", err.Error())
				return 1
			}

			return 0
		})

	config_pid := cli.NewCommand("pid", "set process id (lower pid wins in case of conflicts").
		WithArg(cli.NewArg("id", "the new process id.").WithType(cli.TypeInt)).
		WithAction(func(args []string, options map[string]string) int {
			pid64, err := strconv.ParseInt(args[0], 10, 64)

			if err != nil {
				// should never happen, as cli lib does type checking.
				panic(err)
			}
			if pid64 <= 0 || pid64 > math.MaxUint32 {
				// fmt.Println("PID has to be in [1,", math.MaxUint32, "] (inclusive).") // does not compile for 32 bit targets -.-
				fmt.Println("PID has to be in [1,4294967295] (inclusive).")
				return 1
			}
			pid := uint32(pid64)

			p := getPersistance(options)
			p.State.Pid = pid
			err = p.CommitState()
			if err != nil {
				fmt.Println("Internal error: ", err.Error())
				return 1
			}

			return 0
		})

	config_remote := cli.NewCommand("remote", "set remote server").
		WithArg(cli.NewArg("host", `example: "1.2.3.4:4242", "" means using no remote`)).
		WithAction(func(args []string, options map[string]string) int {
			host := strings.TrimSpace(args[0])

			p := getPersistance(options)

			url, err := url.ParseRequestURI(host)
			if err == nil {
				// url parsed fine, use it
				// ensure it's http(s)
				if !strings.HasPrefix(url.Scheme, "http") {
					url.Scheme = "https"
					fmt.Println("Warning: only http(s) in URLs is supported, defaulting to https: ", url)
				}
				// ensure there's a terminal /
				// note that this doesn't handle the escaped path or the raw path
				if url.Path != "" && !strings.HasSuffix(url.Path, "/") {
					url.Path += "/"
				}
				p.State.RemoteURL = url
			} else {
				// only warn if the user isn't intentionally unsetting the remote
				if host != "" {
					fmt.Println("Invalid url, unsetting remote: ", err.Error())
				}
				p.State.RemoteURL = url
			}

			err = p.CommitState()
			if err != nil {
				fmt.Println("Internal error: ", err.Error())
				return 1
			}

			return 0
		})

	config := cli.NewCommand("config", "set config parameters").
		WithCommand(config_show).
		WithCommand(config_key).
		WithCommand(config_remote).
		WithCommand(config_pid)

	url := cli.NewCommand("url", "show shareable url of an entry").
		WithArg(cli.NewArg("key", "the key of your entry")).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]
			p := getPersistance(options)
			entry, err := p.GetEntry(key)
			if err != nil {
				panic(err)
			}

			if entry == nil {
				logger.Fatal("Key not found.")
			}

			u, _ := p.State.RemoteURL.Parse("get") // this should never fail
			q := u.Query()
			q.Set("q", entry.UrlToken)
			u.RawQuery = q.Encode()
			fmt.Println(u)
			return 0
		})
	qr := cli.NewCommand("qr", "print shareable qr code of entry to console").
		WithArg(cli.NewArg("key", "the key of your entry")).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]
			p := getPersistance(options)
			defer p.Close()

			err := p.GetRemoteUpdates()
			if err != nil {
				logger.Println("Couldn't get updates from server. ", err)
			}
			entry, err := p.GetEntry(key)
			if err != nil {
				panic(err)
			}

			if entry == nil {
				logger.Fatal("Key not found.")
			}

			url, _ := p.State.RemoteURL.Parse("get") // this should never fail
			q := url.Query()
			q.Set("q", entry.UrlToken)
			url.RawQuery = q.Encode()
			qrcode.QRCode(url.String(), qrcode.BrightBlack, qrcode.BrightWhite, qr.Low)

			return 0
		})

	app := cli.New("- a personal KV store").
		WithOption(cli.NewOption("db", "the database file to use (default: ~/.b.sqlite)")).
		WithCommand(ls).
		WithCommand(get).
		WithCommand(set).
		WithCommand(rm).
		WithCommand(url).
		WithCommand(qr).
		WithCommand(config).
		WithCommand(serve)
	return app

}
