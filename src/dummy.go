package kvass

func NewDummyPersistance() *dummyPersistance {
	p := dummyPersistance{}
	p.entries = make(map[string]KvEntry)
	return &p
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
