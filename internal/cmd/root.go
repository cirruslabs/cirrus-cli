package cmd

import (
	"github.com/cirruslabs/cirrus-cli/internal/cmd/validate"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cirrus",
		Short: "Cirrus CLI",
	}

	cmd.AddCommand(validate.NewValidateCmd())

	return cmd
}
