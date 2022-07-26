package kvass

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func RunServer(p *SqlitePersistance, bind string) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

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
	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("q")

		if file == "" {
			http.Error(w, "Please specify file", 400)
			return
		}

		row := p.db.QueryRow("select key from entries where urltoken = ?;", file)
		var key string
		err := row.Scan(&key)
		if err == sql.ErrNoRows {
			http.Error(w, "Unknown File", 404)
			return
		}

		entry, err := p.GetEntry(key)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}

		if entry == nil {
			http.Error(w, "Aaaaand it's gone", 419) // entry was deleted since we got the key
			return
		}

		if strings.HasSuffix(key, ".html") {
			r.Header.Add("Content-Type", "application/html")
		}

		io.Copy(w, bytes.NewBuffer(entry.Value))

	})

	logger.Printf("Server started and listening on %v\n", bind)
	panic(http.ListenAndServe(bind, nil))
}
