//go:build windows

package privdrop

import (
	"fmt"
	"syscall"
)

var SysProcAttr *syscall.SysProcAttr

func Initialize(username string) error {
	return fmt.Errorf("privilege dropping is not supported on this platform")
}
