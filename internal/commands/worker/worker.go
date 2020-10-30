package worker

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Persistent worker mode",
	}

	cmd.AddCommand(
		NewRunCmd(),
	)

	return cmd
}
