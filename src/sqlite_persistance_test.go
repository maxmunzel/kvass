package kvass

import (
	"fmt"
	"testing"
)

func TestUpdates(t *testing.T) {
	t.Parallel()

	p, _ := NewSqlitePersistance(":memory:")
	for i := 0; i < 100; i += 1 {
		err := Set(p, "test", []byte(fmt.Sprint(i)))
		if err != nil {
			panic(err)
		}
		Set(p, "foo", []byte("bar"))
	}

	val, err := p.GetEntry("test")
	if err != nil {
		t.Errorf(err.Error())
	}
	if string(val.Value) != "99" {
		t.Error("key did not properly update")
	}

	val, err = p.GetEntry("foo")
	if err != nil {
		t.Errorf(err.Error())
	}
	if string(val.Value) != "bar" {
		t.Error("key did not properly update")
	}
	val, err = p.GetEntry("nonexistent")
	if err != nil {
		t.Errorf(err.Error())
	}
	if val != nil {
		t.Error("unset key had unexpected value")
	}

}
