package main

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"log"
	"os"
	"os/signal"
)

var (
	version = "unknown"
	commit = "unknown"
)

func main() {
	// Set up signal interruptible context
	ctx, cancel := context.WithCancel(context.Background())

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	go func() {
		select {
		case <-interruptCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// Run the command
	if err := commands.NewRootCmd(version, commit).ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
