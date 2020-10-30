package commands

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/internal"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cirrus",
		Short:   "Cirrus CLI",
		Version: version.FullVersion,
	}

	cmd.AddCommand(
		newValidateCmd(),
		newRunCmd(),
		newServeCmd(),
		internal.NewRootCmd(),
	)

	return cmd
}
