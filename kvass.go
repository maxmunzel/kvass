package main

import (
	kvass "github.com/maxmunzel/kvass/src"
	"os"
)

func main() {
	app := kvass.GetApp()
	os.Exit(app.Run(os.Args, os.Stdout))
}
