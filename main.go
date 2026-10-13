package main

import (
	"os"

	"github.com/streambinder/pango/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
