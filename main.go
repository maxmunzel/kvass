package main

import (
	"encoding/json"
	"fmt"
	kvass "github.com/maxmunzel/kvass/src"
)

func main() {
	p, _ := kvass.NewSqlitePersistance("test.sqlite")
	for i := 0; i < 100; i += 1 {
		err := kvass.Set(p, "test", fmt.Sprint(i))
		if err != nil {
			panic(err)
		}
		v, _ := p.GetValue("test")
		if v != fmt.Sprint(i) {
			fmt.Println(v, "!=", i)
		}
		kvass.Set(p, "foo", "bar")
	}

	fmt.Println(p.GetValue("test"))
	fmt.Println(p.GetValue("foo"))
	fmt.Println(p.GetValue("nonexistent"))

	updates, _ := p.GetUpdates(0)
	entries, err := json.MarshalIndent(updates, "", " ")
	if err != nil {
		panic(err)
	}
	println(string(entries))
	fmt.Println(p)

}
