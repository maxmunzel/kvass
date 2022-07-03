package kvass

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func RunServer(p Persistance, bind string) {
	p.GetProcessID()
	http.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		entries := make([]KvEntry, 0)
		err = json.Unmarshal(payload, &entries)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		for _, e := range entries {
			p.UpdateOn(e)
		}

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
	http.ListenAndServe(bind, nil)
}
