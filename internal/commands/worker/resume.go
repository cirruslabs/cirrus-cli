package worker

import (
	"github.com/spf13/cobra"
)

func NewResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume task scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := buildWorker(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			return worker.Resume(cmd.Context())
		},
	}

	attachFlags(cmd)

	return cmd
}
