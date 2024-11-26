package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/privdrop"
	"github.com/spf13/cobra"
	"runtime"
)

var ErrRun = errors.New("run failed")

var username string

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run persistent worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize privilege dropping on macOS, if requested
			if username != "" {
				if err := privdrop.Initialize(username); err != nil {
					return err
				}
			}

			worker, err := buildWorker(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if err := worker.Run(cmd.Context()); err != nil {
				return fmt.Errorf("%w: %v", ErrRun, err)
			}
			return nil
		},
	}

	// We only need privilege dropping on macOS due to newly introduced
	// "Local Network" permission, which cannot be disabled automatically,
	// and according to the Apple's documentation[1], running Persistent
	// Worker as a superuser is the only choice.
	//
	// Note that the documentation says that "macOS automatically allows
	// local network access by:" and "Any daemon started by launchd". However,
	// this is not true for daemons that have <key>UserName</key> set to non-root.
	//
	//nolint:lll // can't make the link shorter
	// [1]: https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy#macOS-considerations
	if runtime.GOOS == "darwin" {
		cmd.Flags().StringVar(&username, "user", "", "user name to drop privileges to"+
			" when running external programs (e.g. Tart, Vetu, etc.)")
	}

	attachFlags(cmd)

	return cmd
}
