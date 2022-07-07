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

	val, err := p.GetValue("test")
	if err != nil {
		t.Errorf(err.Error())
	}
	if string(val) != "99" {
		t.Error("key did not properly update")
	}

	val, err = p.GetValue("foo")
	if err != nil {
		t.Errorf(err.Error())
	}
	if string(val) != "bar" {
		t.Error("key did not properly update")
	}
	val, err = p.GetValue("nonexistent")
	if err != nil {
		t.Errorf(err.Error())
	}
	if string(val) != "" {
		t.Error("unset key had unexpected value")
	}

}
