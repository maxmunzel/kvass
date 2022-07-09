package kvass

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"modernc.org/mathutil"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
)

type SqliteState struct {
	Counter        uint64
	Pid            uint32
	Key            string
	RemoteHostname string
	RemoteCounter  uint64
}

type SqlitePersistance struct {
	path  string
	db    *sql.DB
	State SqliteState
}

func panicIfNonNil(err error) {
	if err != nil {
		panic(err)
	}
}

func (p *SqlitePersistance) GetRemoteUpdates() (err error) {
	if p.State.RemoteHostname == "" {
		return nil
	}
	request, err := json.Marshal(UpdateRequest{ProcessID: p.State.Pid, Counter: p.State.RemoteCounter})
	if err != nil {
		return err
	}

	request, err = p.Encrypt(request)
	if err != nil {
		return err
	}

	resp, err := http.Post("http://"+p.State.RemoteHostname+"/pull", "application/json", bytes.NewReader(request))
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body, err = p.DecryptData(body)
	if err != nil {
		return err
	}

	updates := make([]KvEntry, 0)
	err = json.Unmarshal(body, &updates)
	if err != nil {
		return err
	}

	for _, u := range updates {
		err = p.UpdateOn(u)
		if err != nil {
			return err
		}
	}
	return nil

}

func (p *SqlitePersistance) Push() error {
	// push changes to remote

	host := p.State.RemoteHostname
	if host == "" {
		return nil
	}
	updates, err := p.GetUpdates(UpdateRequest{Counter: p.State.RemoteCounter, ProcessID: ReservedProcessID})
	if err != nil {
		panic(err)
	}
	payload, err := json.Marshal(updates)
	if err != nil {
		panic(err)
	}
	payload, err = p.Encrypt(payload)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Post("http://"+host+"/push", "application/json", bytes.NewReader(payload))
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Error posting update to server: ", err)
	}
	return nil
}

func (s *SqlitePersistance) CommitState() error {
	// saves the internal state to the sqlite db
	state, err := json.MarshalIndent(&s.State, "", " ")
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()

	defer tx.Rollback()
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE from state;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT into state values (?);", state)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SqlitePersistance) Close() error {
	return s.db.Close()
}

func NewSqlitePersistance(path string) (*SqlitePersistance, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// check if DB has the expected tables

	persistance := &SqlitePersistance{
		path: path,
		db:   db,
	}

	if _, err := os.Stat(path); err == nil {

		// load state and return
		row := db.QueryRow("select * from state;")
		var state_json []byte
		err := row.Scan(&state_json)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(state_json, &persistance.State)
		if err != nil {
			return nil, err
		}

	} else {
		// init DB
		_, err := db.Exec(`
        create table if not exists entries (key, value, timestamp, pid, counter, urltoken); 
        create table if not exists state   (state);`)

		if err != nil {
			return nil, err
		}
		pid_, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
		persistance.State.Pid = uint32(pid_.Int64() + 1)
		if err != nil {
			panic(err)
		}

		// generate key
		key := make([]byte, 32)
		_, err = io.ReadFull(rand.Reader, key)
		if err != nil {
			return nil, err
		}
		persistance.State.Key = hex.EncodeToString(key)

		err = persistance.CommitState()
		if err != nil {
			return nil, err
		}
	}
	return persistance, nil
}

func (s *SqlitePersistance) DecryptData(data []byte) ([]byte, error) {
	key, err := hex.DecodeString(s.State.Key)
	if err != nil {
		return nil, err
	}

	if len(key) != 32 {
		return nil, errors.New("Invalid key length.")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < gcm.NonceSize() {
		return nil, errors.New("Data too short!")
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	return gcm.Open(nil, nonce, ciphertext, nil)
}
func (s *SqlitePersistance) Encrypt(data []byte) ([]byte, error) {
	key, err := hex.DecodeString(s.State.Key)
	if err != nil {
		return nil, err
	}

	if len(key) != 32 {
		return nil, errors.New("Invalid key length.")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	result := append(nonce, ciphertext...)
	return result, nil

}
func (s *SqlitePersistance) GetProcessID() (uint32, error) {
	return s.State.Pid, nil
}
func (s *SqlitePersistance) GetCounter() (uint64, error) {
	return s.State.Counter, nil
}

type UpdateRequest struct {
	Counter   uint64
	ProcessID uint32
}

func (s *SqlitePersistance) GetUpdates(req UpdateRequest) ([]KvEntry, error) {
	result := make([]KvEntry, 0)
	rows, err := s.db.Query("select * from entries where counter >= ? and pid != ?;", req.Counter, req.ProcessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry KvEntry
		err = rows.Scan(&entry.Key, &entry.Value, &entry.TimestampUnixMicro, &entry.ProcessID, &entry.Counter, &entry.UrlToken)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil

}
func (s *SqlitePersistance) UpdateOn(entry KvEntry) error {
	// get current entry from db
	tx, err := s.db.Begin()
	defer tx.Rollback()
	var oldEntry KvEntry
	row := tx.QueryRow("select * from entries order by timestamp desc, pid desc, counter desc where key = ? limit 1;", entry.Key)
	err = row.Scan(&oldEntry.Key, &oldEntry.Value, &oldEntry.TimestampUnixMicro, &oldEntry.ProcessID, &oldEntry.Counter, &oldEntry.UrlToken)
	if err != nil {
		// no result
		oldEntry = entry
	}

	// select LUB of old and new entry
	s.State.RemoteCounter = mathutil.MaxUint64(s.State.RemoteCounter, entry.Counter)

	entry = entry.Max(oldEntry)

	newCounter := mathutil.MaxUint64(entry.Counter, s.State.Counter) + 1

	entry.Counter = newCounter
	s.State.Counter = newCounter
	// write back LUB to db
	_, err = tx.Exec("delete from entries where key = ?;", entry.Key)
	if err != nil {
		return err
	}

	_, err = tx.Exec("insert into entries values (?, ?, ?, ?, ?, ?);",
		entry.Key,
		entry.Value,
		entry.TimestampUnixMicro,
		entry.ProcessID,
		entry.Counter,
		entry.UrlToken,
	)

	err = tx.Commit()
	if err != nil {
		return err
	}
	return s.CommitState()

}
func (s *SqlitePersistance) GetKeys() ([]string, error) {
	result := make([]string, 0)

	rows, err := s.db.Query("select distinct key from entries where length(value) != 0 order by key asc;")
	if err != nil {
		return result, err
	}
	var entry string

	for rows.Next() {
		err := rows.Scan(&entry)
		if err != nil {
			return result, err
		}
		result = append(result, entry)
	}
	return result, nil
}
func (s *SqlitePersistance) GetEntry(key string) (*KvEntry, error) {

	row := s.db.QueryRow("select * from entries where key = ? order by timestamp desc, pid desc, counter desc limit 1;", key)
	entry := KvEntry{}
	err := row.Scan(&entry.Key, &entry.Value, &entry.TimestampUnixMicro, &entry.ProcessID, &entry.Counter, &entry.UrlToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}
