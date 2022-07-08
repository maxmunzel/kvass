package kvass

import (
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

	err = p.UpdateOn(KvEntry{
		ProcessID:          pid,
		TimestampUnixMicro: t,
		Key:                key,
		Value:              value,
		Counter:            count,
	})
	return err

}
