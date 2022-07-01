package main

import (
	kvass "github.com/maxmunzel/kvass/src"
)

func main() {
	p := kvass.NewDummyPersistance()
	kvass.RunServer(p)
}
