//go:build !darwin
// +build !darwin

package parallels

import (
	"fmt"
	"runtime"
)

func ensureNoVMsRunning() error {
	return fmt.Errorf("%w: unsupported platform: %s", ErrVMKiller, runtime.GOOS)
}
