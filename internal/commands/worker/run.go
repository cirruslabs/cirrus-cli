package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/localnetworkhelper"
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
			// Drop the privileges, if requested
			if username != "" {
				// Additionally, when running on macOS, start the macOS "Local
				// Network" permission helper and establish a connection with it
				if runtime.GOOS == "darwin" {
					if err := localnetworkhelper.StartAndConnect(cmd.Context()); err != nil {
						return err
					}
				}

				if err := privdrop.Drop(username); err != nil {
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

	cmd.Flags().StringVar(&username, "user", "", "user name to drop privileges to")

	attachFlags(cmd)

	return cmd
}
