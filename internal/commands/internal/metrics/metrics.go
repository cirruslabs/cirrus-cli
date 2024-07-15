package metrics

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	logger := logrus.New()

	resultChan := metrics.Run(cmd.Context(), logger)

	result := <-resultChan

	if len(result.Errors()) != 0 {
		for _, err := range result.Errors() {
			logrus.Fatalf("metrics failed: %v", err)
		}
	}

	return nil
}
