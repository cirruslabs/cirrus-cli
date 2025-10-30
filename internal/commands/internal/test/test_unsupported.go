//go:build !linux && !darwin && !windows

package test

import "github.com/spf13/cobra"

func NewTestCmd() *cobra.Command {
	return nil
}
