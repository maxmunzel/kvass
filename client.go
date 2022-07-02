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
	p := kvass.NewDummyPersistance()
	return p
}
func main() {
	logger := log.New(os.Stderr, "", log.Llongfile|log.LstdFlags)
	get := cli.NewCommand("get", "get a value").
		WithArg(cli.NewArg("key", "the key to get")).
		WithAction(func(args []string, options map[string]string) int {
			key := args[0]
			p := getPersistance(options)
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
				logger.Panicln("Error posting update to server: ", err.Error())
				return 1
			}
			return 0
		})
	app := cli.New("kvass - a personal KV store").
		WithArg(cli.NewArg("host", "the server to sync with").AsOptional()).
		WithCommand(get).
		WithCommand(set)

	os.Exit(app.Run(os.Args, os.Stdout))

}
