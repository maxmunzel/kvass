package kvass

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func RunServer(p *SqlitePersistance, bind string) {
	p.GetProcessID()
	http.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		payload, err = p.DecryptData(payload)
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
		payload_enc, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		payload, err := p.DecryptData(payload_enc)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		updateRequest := UpdateRequest{}
		err = json.Unmarshal(payload, &updateRequest)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		updates, err := p.GetUpdates(updateRequest)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		response_payload, err := json.MarshalIndent(updates, "", " ")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		response_payload, err = p.Encrypt(response_payload)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write(response_payload)
	})
	panic(http.ListenAndServe(bind, nil))
}
