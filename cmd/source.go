package main

import (
	"os"

	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/replace"
)

func main() {
	os.Exit(sourcerer.Source("replaceCLI", []sourcerer.CLI{replace.CLI()}))
}
