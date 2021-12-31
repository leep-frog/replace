package main

import (
	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/replace"
)

func main() {
	sourcerer.Source(replace.CLI())
}
