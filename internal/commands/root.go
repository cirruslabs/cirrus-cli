package commands

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/internal"
	"github.com/cirruslabs/cirrus-cli/internal/commands/worker"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cirrus",
		Short:   "Cirrus CLI",
		Version: version.FullVersion,
	}

	commands := []*cobra.Command{
		newValidateCmd(),
		newRunCmd(),
		newServeCmd(),
		internal.NewRootCmd(),
		worker.NewRootCmd(),
	}

	return helpers.ConsumeSubCommands(cmd, commands)
}
