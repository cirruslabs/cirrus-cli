package internal

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/internal/test"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "internal",
		Short:  "CLI development commands",
		Hidden: true,
	}

	commands := []*cobra.Command{
		test.NewTestCmd(),
	}

	return helpers.ConsumeSubCommands(cmd, commands)
}
