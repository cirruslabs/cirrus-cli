package internal

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/internal/test"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "internal",
		Short:  "CLI development commands",
		Hidden: true,
	}

	cmd.AddCommand(
		test.NewTestCmd(),
	)

	return cmd
}
