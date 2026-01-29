package metrics

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/spf13/cobra"
	"log/slog"
)

func NewMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Debug agent metrics collection routines",
		RunE:  run,
	}

	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	logger := slog.Default()

	resultChan := metrics.Run(cmd.Context(), logger)

	result := <-resultChan

	if len(result.Errors()) != 0 {
		for _, err := range result.Errors() {
			return fmt.Errorf("metrics failed: %w", err)
		}
	}

	return nil
}
