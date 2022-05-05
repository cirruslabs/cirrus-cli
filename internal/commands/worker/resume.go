//go:build !windows

package worker

import (
	"github.com/spf13/cobra"
)

func NewResumeCmd() *cobra.Command {
	flags := &workerConfig{}

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume task scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := flags.buildWorker(cmd)
			if err != nil {
				return err
			}
			return worker.Resume(cmd.Context())
		},
	}

	flags.attacheFlags(cmd)

	return cmd
}
