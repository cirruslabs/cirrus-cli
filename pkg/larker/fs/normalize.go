//go:build !windows
// +build !windows

package fs

import "syscall"

var ErrNormalizedIsADirectory = syscall.EISDIR
