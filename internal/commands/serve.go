//go:build !windows
// +build !windows

package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/cirruslabs/cirrus-cli/internal/evaluator"
	"github.com/cirruslabs/cirrus-cli/internal/logginglevel"
	"github.com/spf13/cobra"
	slogctx "github.com/veqryn/slog-context"
)

var ErrServe = errors.New("serve failed")

var address string

func serve(cmd *cobra.Command, args []string) error {
	// Initialize logger: produce machine-friendly output
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logginglevel.Level,
	})
	// Initialize logger: support context.Context
	slogctxHandler := slogctx.NewHandler(jsonHandler, &slogctx.HandlerOptions{})
	// Initialize logger: final steps
	logger := slog.New(slogctxHandler)
	slog.SetDefault(logger)

	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrServe, err)
	}

	fmt.Printf("listening on %s\n", lis.Addr().String())

	if err := evaluator.Serve(cmd.Context(), lis); err != nil {
		return fmt.Errorf("%w: %v", ErrServe, err)
	}

	return nil
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "serve [flags]",
		Short:  "Run RPC server that evaluates YAML and Starlark configurations",
		RunE:   serve,
		Hidden: true,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cmd.PersistentFlags().StringVarP(&address, "listen", "l", fmt.Sprintf(":%s", port), "address to listen on")

	return cmd
}
