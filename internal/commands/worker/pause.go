package worker

import (
	"github.com/spf13/cobra"
)

var waitForTasksToFinish bool

func NewPauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause task scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := buildWorker(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			return worker.Pause(cmd.Context(), waitForTasksToFinish)
		},
	}

	attachFlags(cmd)

	cmd.PersistentFlags().BoolVar(&waitForTasksToFinish, "wait", false, "wait for currently running tasks to finish")

	return cmd
}
