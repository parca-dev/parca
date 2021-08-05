package main

import (
	"os"

	"github.com/parca-dev/parca/cmd/parca/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
