package commands

import "github.com/spf13/cobra"

var version string

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cirrus",
		Short:   "Cirrus CLI",
		Version: version,
	}

	cmd.AddCommand(
		newValidateCmd(),
		newRunCmd(),
	)

	return cmd
}
