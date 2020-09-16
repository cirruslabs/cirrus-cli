package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"runtime/debug"
)

func NewRootCmd(version, commit string) *cobra.Command {
	if version == "unknown" {
		info, ok := debug.ReadBuildInfo()
		if ok {
			version = info.Main.Version
		}
	}

	cmd := &cobra.Command{
		Use:     "cirrus",
		Short:   "Cirrus CLI",
		Version: fmt.Sprintf("%s-%s", version, commit),
	}

	cmd.AddCommand(
		newValidateCmd(),
		newRunCmd(),
	)

	return cmd
}
