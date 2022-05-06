package worker

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Persistent worker mode",
	}

	commands := []*cobra.Command{
		NewRunCmd(),
		NewPauseCmd(),
		NewResumeCmd(),
	}

	return helpers.ConsumeSubCommands(cmd, commands)
}
