package main

import (
	"github.com/cirruslabs/cirrus-cli/internal/cmd"
	"log"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
