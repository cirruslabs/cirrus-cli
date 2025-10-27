package commands

import (
	"log/slog"

	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/internal"
	"github.com/cirruslabs/cirrus-cli/internal/commands/localnetworkhelper"
	"github.com/cirruslabs/cirrus-cli/internal/commands/validate"
	"github.com/cirruslabs/cirrus-cli/internal/commands/worker"
	"github.com/cirruslabs/cirrus-cli/internal/logginglevel"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/spf13/cobra"
)

var debug bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cirrus",
		Short:         "Cirrus CLI",
		Version:       version.FullVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if debug {
				logginglevel.Level.Set(slog.LevelDebug)
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

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
