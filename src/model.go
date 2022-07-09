package kvass

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

type ValueType = []byte

const ReservedProcessID = 0

type KvEntry struct {
	Key                string
	Value              ValueType
	TimestampUnixMicro int64
	ProcessID          uint32
	Counter            uint64
	UrlToken           string
}

func (e KvEntry) isGreaterOrEqualThan(other KvEntry) bool {
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

	if e.Counter > other.Counter {
		return true
	}
	return false
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
func Set(p *SqlitePersistance, key string, value ValueType) error {
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
	})
	return err

}
