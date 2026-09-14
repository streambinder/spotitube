package main

import (
	"os"

	"github.com/streambinder/spotitube/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
