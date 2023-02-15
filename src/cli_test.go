package kvass

import (
	"bytes"
	"io"
	"os"
	"testing"
)

const SERVER_PORT = "34521"
const REMOTE_DB = "test_remote.sqlite"
const LOCAL_DB = "test_local.sqlite"
const KEY = "cd9994e3a0c5ca40c2652ff21ab401d6d3a1a5c5fd5482ae62c03ce32d340b49"

func TestCli(t *testing.T) {
	fail := func(err error) {
		if err != nil {
			t.FailNow()
		}
	}

	assertKeyEqualsVal := func(db string, key string, value []byte) {
		instance, err := NewSqlitePersistance(db)
		fail(err)

		instanceEntry, err := instance.GetEntry(key)
		fail(err)
		if instanceEntry == nil {
			t.Errorf("%s instance did not have a value for key %s.", db, key)
		} else if !bytes.Equal(value, instanceEntry.Value) {
			t.Errorf("%s instance had unexpected value for key %s. \nExpected: '%s'\nActual: '%s'", db, key, value, instanceEntry.Value)
		}
	}
	os.Remove(LOCAL_DB)
	os.Remove(REMOTE_DB)
	t.Cleanup(func() { os.Remove(REMOTE_DB) })
	t.Cleanup(func() { os.Remove(LOCAL_DB) })

	for _, db := range []string{LOCAL_DB, REMOTE_DB} {
		GetApp().Run([]string{
			"kvass",
			"config",
			"key",
			"--db=" + db,
			KEY,
		}, io.Discard)
	}

	// TODO set keys
	go GetApp().Run([]string{
		"kvass",
		"serve",
		"--db=" + REMOTE_DB,
		"--bind=localhost:" + SERVER_PORT,
	}, io.Discard)

	GetApp().Run([]string{
		"kvass",
		"config",
		"remote",
		"--db=" + LOCAL_DB,
		"http://localhost:" + SERVER_PORT,
	}, io.Discard)

	GetApp().Run([]string{
		"kvass",
		"set",
		"--db=" + LOCAL_DB,
		"testkey",
		"testval",
	}, io.Discard)

	assertKeyEqualsVal(LOCAL_DB, "testkey", []byte("testval\n"))
	assertKeyEqualsVal(REMOTE_DB, "testkey", []byte("testval\n"))

	GetApp().Run([]string{
		"kvass",
		"set",
		"--db=" + REMOTE_DB,
		"testkey",
		"OVERWRITTEN",
	}, io.Discard)

	GetApp().Run([]string{
		"kvass",
		"get",
		"--db=" + LOCAL_DB,
		"testkey",
	}, io.Discard)

	assertKeyEqualsVal(LOCAL_DB, "testkey", []byte("OVERWRITTEN\n"))
	assertKeyEqualsVal(REMOTE_DB, "testkey", []byte("OVERWRITTEN\n"))

	//remote, err := NewSqlitePersistance(REMOTE_DB)
	//fail(err)

	//remoteEntry, err := remote.GetEntry("testkey")
	//fail(err)

}
