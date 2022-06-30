package main

import (
	"encoding/json"
	"fmt"
	kvass "github.com/maxmunzel/kvass/src"
	"io/ioutil"
	"net/http"
)

func main() {
	p := kvass.NewDummyPersistance()
	p.GetProcessID()
	http.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		entries := make([]kvass.KvEntry, 0)
		err = json.Unmarshal(payload, &entries)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		for _, e := range entries {
			p.UpdateOn(e)
		}

		fmt.Println(p)

	})
	http.HandleFunc("/pull", func(w http.ResponseWriter, r *http.Request) {
		updates, err := p.GetUpdates(0)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		payload, err := json.MarshalIndent(updates, "", " ")

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write(payload)
	})
	http.ListenAndServe("127.0.0.1:8000", nil)
}
