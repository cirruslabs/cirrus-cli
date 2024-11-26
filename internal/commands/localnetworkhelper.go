package commands

import (
	"github.com/cirruslabs/cirrus-cli/pkg/localnetworkhelper"
	"github.com/spf13/cobra"
)

func newLocalNetworkHelperCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    localnetworkhelper.CommandName,
		Short:  "Run the macOS \"Local Network\" permission helper process",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// When we start the macOS "Local Network" permission helper process,
			// we pass it one of the socketpair(2)'s descriptors through ExtraFiles
			// field of Golang's exec.Cmd[1]. Here it becomes FD number 3, according
			// to the ExtraFiles documentation:
			//
			// >If non-nil, entry i becomes file descriptor 3+i.
			//
			// [1]: https://pkg.go.dev/os/exec#Cmd
			return localnetworkhelper.Serve(cmd.Context(), 3)
		},
	}
	return cmd
}
