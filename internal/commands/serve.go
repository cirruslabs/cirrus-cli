//go:build !windows
// +build !windows

package commands

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/evaluator"
	"github.com/spf13/cobra"
	"net"
	"os"
)

var ErrServe = errors.New("serve failed")

var address string

func serve(cmd *cobra.Command, args []string) error {
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
