package commands

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cirrus",
		Short: "Cirrus CLI",
	}

	cmd.AddCommand(newValidateCmd())

	return cmd
}
