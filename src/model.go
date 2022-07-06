package kvass

import (
	"time"
)

type ValueType = []byte

type KvEntry struct {
	Key                string
	Value              ValueType
	TimestampUnixMicro int64
	ProcessID          int
	Counter            int64
}

func (e KvEntry) isGreaterOrEqualThan(other KvEntry) bool {
	if e.Counter > other.Counter {
		return true
	}
	if e.Counter < other.Counter {
		return false
	}
	if e.TimestampUnixMicro > other.TimestampUnixMicro {
		return true
	}
	if e.TimestampUnixMicro < other.TimestampUnixMicro {
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

type Persistance interface {
	GetProcessID() (int, error)
	UpdateOn(entry KvEntry) error
	GetUpdates(startUnixMicros int64) ([]KvEntry, error)
	GetValue(key string) (ValueType, error)
	GetCounter() (int64, error)
	Close() error
}

func Set(p Persistance, key string, value ValueType) error {
	pid, err := p.GetProcessID()
	if err != nil {
		return err
	}
	t := time.Now().UnixMicro()

	count, err := p.GetCounter()
	if err != nil {
		return err
	}

	err = p.UpdateOn(KvEntry{
		ProcessID:          pid,
		TimestampUnixMicro: t,
		Key:                key,
		Value:              value,
		Counter:            count,
	})
	return err

}
