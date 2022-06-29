package kvass

import (
	"time"
)

type ValueType = string

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
}

type dummyPersistance struct {
	pid     int
	counter int64
	entries map[string]KvEntry
}

func (d *dummyPersistance) GetCounter() (int64, error) {
	return d.counter, nil
}

func (d *dummyPersistance) GetProcessID() (int, error) {
	return d.pid, nil
}

func (d *dummyPersistance) GetValue(key string) (ValueType, error) {
	return d.entries[key].Value, nil
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (d *dummyPersistance) UpdateOn(entry KvEntry) error {
	d.counter = max(d.counter, entry.Counter) + 1
	entry.Counter = d.counter
	d.entries[entry.Key] = entry.Max(d.entries[entry.Key])
	return nil
}

func (d *dummyPersistance) GetUpdates(time int64) ([]KvEntry, error) {
	result := make([]KvEntry, 0)
	for _, v := range d.entries {
		if v.TimestampUnixMicro >= time {
			result = append(result, v)
		}
	}
	return result, nil
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

func NewDummyPersistance() *dummyPersistance {
	p := dummyPersistance{}
	p.entries = make(map[string]KvEntry)
	return &p
}
