package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/chacha/pkg/localnetworkhelper"
	"github.com/cirruslabs/chacha/pkg/privdrop"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	oldprivdrop "github.com/cirruslabs/cirrus-cli/pkg/privdrop"
	"github.com/spf13/cobra"
	"runtime"
)

var ErrRun = errors.New("run failed")

var username string
var userCoarseGrained bool

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run persistent worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts []worker.Option

			// Run the macOS "Local Network" permission helper
			// when privilege dropping is requested
			if username != "" {
				if userCoarseGrained {
					if err := oldprivdrop.Initialize(username); err != nil {
						return err
					}
				} else {
					localNetworkHelper, err := localnetworkhelper.New(cmd.Context())
					if err != nil {
						return err
					}

					opts = append(opts, worker.WithLocalNetworkHelper(localNetworkHelper))

					if err := privdrop.Drop(username); err != nil {
						return err
					}
				}
			}

			worker, err := buildWorker(cmd.ErrOrStderr(), opts...)
			if err != nil {
				return err
			}
			if err := worker.Run(cmd.Context()); err != nil {
				return fmt.Errorf("%w: %v", ErrRun, err)
			}
			if err := worker.Close(); err != nil {
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
		cmd.Flags().StringVar(&username, "user", "", "username to drop privileges to "+
			"(\"Local Network\" permission workaround: requires starting \"cirrus worker run\" as \"root\", the privileges "+
			"will be then dropped to the specified user after starting the \"cirrus localnetworkhelper\" helper process)")

		cmd.Flags().BoolVar(&userCoarseGrained, "user-coarse-grained", false, "use older, coarse-grained "+
			"privilege dropping mechanism that only applies to external programs (and only Tart, Parallels and unset isolation)")
	}

	attachFlags(cmd)

	return cmd
}
