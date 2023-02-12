package kvass

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

const ReservedProcessID = 0

type KvEntry struct {
	Key      string
	Value    []byte // empty slice means key deleted
	UrlToken string // random token used for url

	// The following fields are used for state merging
	TimestampUnixMicro int64
	ProcessID          uint32 // randomly chosen for each node
	Counter            uint64 // lamport clock
	NeedsToBePushed    bool
}

func (e KvEntry) isGreaterOrEqualThan(other KvEntry) bool {
	// compares two KvEntrys based on their (time, counter, - pid) tuple

	if e.TimestampUnixMicro > other.TimestampUnixMicro {
		return true
	}
	if e.TimestampUnixMicro < other.TimestampUnixMicro {
		return false
	}
	if e.Counter > other.Counter {
		return true
	}
	if e.Counter < other.Counter {
		return false
	}

	if e.ProcessID < other.ProcessID {
		return true
	}

	if e.ProcessID > other.ProcessID {
		return false
	}

	// non of the fields differ -> they are equal
	return true
}

func (e KvEntry) Max(other KvEntry) KvEntry {
	if e.isGreaterOrEqualThan(other) {
		return e
	}
	return other

}

func Delete(p *SqlitePersistance, key string) error {
	return Set(p, key, []byte(""))
}
func Set(p *SqlitePersistance, key string, value []byte) error {
	pid, err := p.GetProcessID()
	if err != nil {
		return err
	}
	t := time.Now().UnixMicro()

	count, err := p.GetCounter()
	if err != nil {
		return err
	}

	url_bytes := make([]byte, 16)
	_, err = rand.Read(url_bytes)
	if err != nil {
		panic(err)
	}

	err = p.UpdateOn(KvEntry{
		ProcessID:          pid,
		TimestampUnixMicro: t,
		Key:                key,
		Value:              value,
		Counter:            count,
		UrlToken:           base64.RawURLEncoding.EncodeToString(url_bytes),
		NeedsToBePushed:    true,
	})
	return err

}
