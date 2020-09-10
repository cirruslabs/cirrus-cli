package commands

import (
	"github.com/spf13/cobra"
	"runtime/debug"
)

var version string

func NewRootCmd() *cobra.Command {
	if version == "" {
		info, ok := debug.ReadBuildInfo()
		if ok && info != nil {
			version = info.Main.Version
		}
	}

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
