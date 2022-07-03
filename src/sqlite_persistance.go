package kvass

import (
	"database/sql"
	"encoding/json"
	_ "modernc.org/sqlite"
	"os"
)

type SqlitePersistance struct {
	path  string
	db    *sql.DB
	state struct {
		Counter int64
		Pid     int
	}
}

func panicIfNonNil(err error) {
	if err != nil {
		panic(err)
	}
}

func (s *SqlitePersistance) commitState() error {
	// saves the internal state to the sqlite db
	state, err := json.MarshalIndent(&s.state, "", " ")
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

	if _, err := os.Stat("path"); err == nil {

		// load state and return
		row := db.QueryRow("select * from state;")
		var state_json []byte
		err := row.Scan(&state_json)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(state_json, &persistance.state)
		if err != nil {
			return nil, err
		}

	} else {
		_, err := db.Exec(`
        create table if not exists entries (key, value, timestamp, pid, counter); 
        create table if not exists state   (state);`)

		if err != nil {
			return nil, err
		}
		err = persistance.commitState()
		if err != nil {
			return nil, err
		}
	}
	return persistance, nil
}

func (s *SqlitePersistance) GetProcessID() (int, error) {
	return s.state.Pid, nil
}
func (s *SqlitePersistance) GetCounter() (int64, error) {
	return s.state.Counter, nil
}
func (s *SqlitePersistance) GetUpdates(time int64) ([]KvEntry, error) {
	result := make([]KvEntry, 0)
	rows, err := s.db.Query("select * from entries where timestamp >= ?;", time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry KvEntry
		err = rows.Scan(&entry.Key, &entry.Value, &entry.TimestampUnixMicro, &entry.ProcessID, &entry.Counter)
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
	row := tx.QueryRow("select * from entries order by timestamp desc, pid desc, counter desc limit 1;")
	err = row.Scan(&oldEntry.Key, &oldEntry.Value, &oldEntry.TimestampUnixMicro, &oldEntry.ProcessID, &oldEntry.Counter)
	if err != nil {
		// no result
		oldEntry = entry
	}

	// select LUB of old and new entry

	entry = entry.Max(oldEntry)

	// write back LUB to db
	_, err = tx.Exec("delete from entries where key = ?;", entry.Key)
	if err != nil {
		return err
	}

	_, err = tx.Exec("insert into entries values (?, ?, ?, ?,?);",
		entry.Key,
		entry.Value,
		entry.TimestampUnixMicro,
		entry.ProcessID,
		entry.Counter,
	)

	return tx.Commit()
}
func (s *SqlitePersistance) GetValue(key string) (ValueType, error) {
	row := s.db.QueryRow("select value from entries where key=? order by timestamp desc, pid desc, counter desc limit 1;", key)
	var value ValueType
	err := row.Scan(&value)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}
	return value, nil
}
