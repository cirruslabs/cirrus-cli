package worker

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
)

var ErrRun = errors.New("run failed")

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run persistent worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := buildWorker(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if err := worker.Run(cmd.Context()); err != nil {
				return fmt.Errorf("%w: %v", ErrRun, err)
			}
			return nil
		},
	}

	attachFlags(cmd)

	return cmd
}
