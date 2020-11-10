package helpers

import "github.com/spf13/cobra"

func ConsumeSubCommands(cmd *cobra.Command, subCommands []*cobra.Command) *cobra.Command {
	var hasValidSubcommands bool

	for _, subCommand := range subCommands {
		if cmd == nil {
			continue
		}

		cmd.AddCommand(subCommand)
		hasValidSubcommands = true
	}

	if hasValidSubcommands {
		return cmd
	}

	return nil
}
