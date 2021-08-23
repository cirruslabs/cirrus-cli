//go:build !linux && !darwin && !windows
// +build !linux,!darwin,!windows

package commands

import "github.com/spf13/cobra"

func newRunCmd() *cobra.Command {
	return nil
}
