package kvass

import (
	"bytes"
	"testing"
)

func TestPush(t *testing.T) {
	// create two instances client and remote, set a key on the client,
	// apply its updates on the remote and check if the remote also set the
	// key.
	t.Parallel()
	fail := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	client, err := NewSqlitePersistance(":memory:")
	defer client.Close()
	fail(err)

	remote, err := NewSqlitePersistance(":memory:")
	fail(err)
	defer remote.Close()

	Set(client, "foo", []byte("bar"))

	updates, err := client.GetUpdates(UpdateRequest{
		Counter:   client.State.RemoteCounter,
		ProcessID: ReservedProcessID,
	})
	fail(err)

	for _, update := range updates {
		fail(remote.UpdateOn(update))
	}

	entry, err := remote.GetEntry("foo")
	fail(err)

	if !bytes.Equal(entry.Value, []byte("bar")) {
		t.Error("Remote did not get the correct value.")
	}

}
