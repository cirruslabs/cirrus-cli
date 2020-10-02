package internal

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "internal",
		Short:  "CLI development commands",
		Hidden: true,
	}

	cmd.AddCommand(
		newTestCmd(),
	)

	return cmd
}
