package main

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"log"
)

func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
