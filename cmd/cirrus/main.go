package main

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/getsentry/sentry-go"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	// Initialize Sentry
	err := sentry.Init(sentry.ClientOptions{
		AttachStacktrace: true,
	})
	if err != nil {
		log.Fatalf("failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

	// Enrich future events with Cirrus CI-specific tags
	if tags, ok := os.LookupEnv("CIRRUS_SENTRY_TAGS"); ok {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			for _, tag := range strings.Split(tags, ",") {
				splits := strings.SplitN(tag, "=", 2)
				if len(splits) != 2 {
					continue
				}

				scope.SetTag(splits[0], splits[1])
			}
		})
	}

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
	if err := commands.NewRootCmd().ExecuteContext(ctx); err != nil {
		// Capture the error into Sentry
		sentry.CaptureException(err)
		sentry.Flush(2 * time.Second)

		// Capture the error into stderr and terminate
		//nolint:gocritic // "log.Fatal will exit, and `defer sentry.Recover()` will not run" — it's OK,
		// since we're already capturing the error above
		log.Fatal(err)
	}
}
