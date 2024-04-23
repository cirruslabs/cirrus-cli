//go:build !windows

package signalfilter

import (
	"os"
	"syscall"
)

// IsNoisy determines which signals provide too much noise both for the text logs
// and the RPC with little debugging value.
func IsNoisy(sig os.Signal) bool {
	return sig == syscall.SIGURG || sig == syscall.SIGCHLD
}
