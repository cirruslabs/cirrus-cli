package commands

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/internal"
	"github.com/cirruslabs/cirrus-cli/internal/commands/localnetworkhelper"
	"github.com/cirruslabs/cirrus-cli/internal/commands/validate"
	"github.com/cirruslabs/cirrus-cli/internal/commands/worker"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cirrus",
		Short:         "Cirrus CLI",
		Version:       version.FullVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	commands := []*cobra.Command{
		validate.NewValidateCmd(),
		newRunCmd(),
		newServeCmd(),
		internal.NewRootCmd(),
		worker.NewRootCmd(),
		localnetworkhelper.NewCommand(),
	}

	return helpers.ConsumeSubCommands(cmd, commands)
}
