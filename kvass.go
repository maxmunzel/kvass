package main

import (
	"bytes"
	"encoding/json"
	kvass "github.com/maxmunzel/kvass/src"
	"github.com/teris-io/cli"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

func GetUpdatesFrom(hostname string) (result []kvass.KvEntry, err error) {
	resp, err := http.Get(hostname)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//result := make([]kvass.KvEntry, 0, 100)
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil

}
func getPersistance(options map[string]string) kvass.Persistance {
	dbpath, contains := options["db"]
	if !contains {

		defaultFilename := ".kvassdb.sqlite"
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		dbpath = path.Join(home, defaultFilename)
	}

	p, err := kvass.NewSqlitePersistance(dbpath)
	if err != nil {
		panic(err)
	}
	return p
}
func main() {
	logger := log.New(os.Stderr, "", log.Llongfile|log.LstdFlags)
	get := cli.NewCommand("get", "get a value").
		WithArg(cli.NewArg("key", "the key to get")).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]
			p := getPersistance(options)
			defer p.Close()
			updates, err := GetUpdatesFrom("http://localhost:8000/pull")
			if err != nil {
				logger.Println("Couldn't get updates from server. ", err.Error())
			} else {

				for _, u := range updates {
					err = p.UpdateOn(u)
					if err != nil {
						panic(err)
					}
				}
			}
			val, err := p.GetValue(key)
			if err != nil {
				panic(err)
			}
			println(string(val))
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
			var val string

			if len(args) < 2 {
				valBytes, err := ioutil.ReadAll(os.Stdin)
				val = string(valBytes)
				if err != nil {
					panic(err)
				}

			} else {
				val = args[1]
			}

			err = kvass.Set(p, key, val)
			if err != nil {
				panic(err)
			}
			updates, err := p.GetUpdates(0)
			if err != nil {
				panic(err)
			}
			payload, err := json.Marshal(updates)
			if err != nil {
				panic(err)
			}

			resp, err := http.DefaultClient.Post("http://localhost:8000/push", "application/json", bytes.NewReader(payload))
			if err != nil || resp.StatusCode != 200 {
				logger.Println("Error posting update to server: ", err.Error())
				return 1
			}
			return 0
		})

	serve := cli.NewCommand("serve", "start in server mode").
		WithOption(cli.NewOption("bind", "bind address (default: localhost:8000)")).
		WithAction(func(args []string, options map[string]string) int {
			bind, contains := options["bind"]
			if !contains {
				bind = "127.0.0.1:8000"
			}
			p := getPersistance(options)
			defer p.Close()
			kvass.RunServer(p, bind)
			return 0
		})

	app := cli.New("kvass - a personal KV store").
		WithArg(cli.NewArg("host", "the server to sync with").AsOptional()).
		WithOption(cli.NewOption("db", "the database file to use (default: ~/.kvassdb.sqlite")).
		WithCommand(get).
		WithCommand(set).
		WithCommand(serve)
	os.Exit(app.Run(os.Args, os.Stdout))

}
