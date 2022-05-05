//go:build !windows

package worker

import (
	"github.com/spf13/cobra"
)

var waitForTasksToFinish bool

func NewPauseCmd() *cobra.Command {
	flags := &workerConfig{}

	cmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause task scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := flags.buildWorker(cmd)
			if err != nil {
				return err
			}
			return worker.Pause(cmd.Context(), waitForTasksToFinish)
		},
	}

	flags.attacheFlags(cmd)

	cmd.PersistentFlags().BoolVar(&waitForTasksToFinish, "wait", false, "wait for currently running tasks to finish")

	return cmd
}
