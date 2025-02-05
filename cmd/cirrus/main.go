package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/breml/rootcerts/embedded"
	"github.com/cirruslabs/cirrus-cli/internal/agent"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/opentelemetry"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/getsentry/sentry-go"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Provide fallback root CA certificates
	mozillaRoots := x509.NewCertPool()
	mozillaRoots.AppendCertsFromPEM([]byte(embedded.MozillaCACertificatesPEM()))
	x509.SetFallbackRoots(mozillaRoots)

	// Initialize Sentry
	var release string

	if version.Version != "unknown" {
		release = fmt.Sprintf("cirrus-cli@%s", version.Version)
	}

	err := sentry.Init(sentry.ClientOptions{
		Release:          release,
		AttachStacktrace: true,
	})
	if err != nil {
		log.Fatalf("failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(5 * time.Second)
	defer sentry.Recover()

	// Run the Cirrus CI Agent if requested
	if len(os.Args) >= 2 && os.Args[1] == "agent" {
		agent.Run(os.Args[2:])

		return
	}

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
	signal.Notify(interruptCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-interruptCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := mainImpl(ctx); err != nil {
		// Capture the error into Sentry
		sentry.CaptureException(err)
		sentry.Flush(2 * time.Second)

		// Capture the error into stderr
		_, _ = fmt.Fprintln(os.Stderr, err)

		// Terminate
		exitCode := 1

		var exitCodeError helpers.ExitCodeError
		if errors.As(err, &exitCodeError) {
			exitCode = exitCodeError.ExitCode()
		}

		//nolint:gocritic // "os.Exit will exit, and `defer sentry.Recover()` will not run" — it's OK,
		// since we're already capturing the error above
		os.Exit(exitCode)
	}
}

func mainImpl(ctx context.Context) error {
	// Initialize OpenTelemetry
	opentelemetryDeinit, err := opentelemetry.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}
	defer opentelemetryDeinit()

	// Run the command
	return commands.NewRootCmd().ExecuteContext(ctx)
}
