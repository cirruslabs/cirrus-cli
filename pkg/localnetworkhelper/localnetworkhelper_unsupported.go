//go:build windows

package localnetworkhelper

import (
	"context"
	"fmt"
)

func StartAndConnect(ctx context.Context) error {
	return fmt.Errorf("macOS \"Local Network\" helper is not supported on this platform")
}

func Serve(ctx context.Context, fd int) error {
	return fmt.Errorf("macOS \"Local Network\" helper is not supported on this platform")
}
