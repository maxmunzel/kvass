package kvass

import (
	"fmt"
	"testing"
)

func TestUpdates(t *testing.T) {
	t.Parallel()

	p, _ := NewSqlitePersistance(":memory:")
	for i := 0; i < 100; i += 1 {
		err := Set(p, "test", fmt.Sprint(i))
		if err != nil {
			panic(err)
		}
		Set(p, "foo", "bar")
	}

	val, err := p.GetValue("test")
	if err != nil {
		t.Errorf(err.Error())
	}
	if val != "99" {
		t.Error("key did not properly update")
	}

	val, err = p.GetValue("foo")
	if err != nil {
		t.Errorf(err.Error())
	}
	if val != "bar" {
		t.Error("key did not properly update")
	}
	val, err = p.GetValue("nonexistent")
	if err != nil {
		t.Errorf(err.Error())
	}
	if val != "" {
		t.Error("unset key had unexpected value")
	}

	p.GetUpdates(0)

}