package commands

import (
	"fmt"
	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"runtime/debug"
)

var (
	version = "unknown"
	commit  = "unknown"
)

func NewRootCmd() *cobra.Command {
	if version == "unknown" {
		info, ok := debug.ReadBuildInfo()
		if ok {
			// We parse the version here for two reasons:
			// * to weed out the "(devel)" version and fallback to "unknown" instead
			//   (see https://github.com/golang/go/issues/29228 for details on when this might happen)
			// * to remove the "v" prefix from the BuildInfo's version (e.g. "v0.7.0") and thus be consistent
			//   with the binary builds, where the version string would be "0.7.0" instead
			semver, err := goversion.NewSemver(info.Main.Version)
			if err == nil {
				version = semver.String()
			}
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
		newServeCmd(),
	)

	return cmd
}
