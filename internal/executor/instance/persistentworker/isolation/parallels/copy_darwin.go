package parallels

import (
	"golang.org/x/sys/unix"
)

func CopyDir(sourceDir string, destinationDir string) error {
	// From clonefile(2) macOS manual page:
	// "If src names a directory, the directory hierarchy is cloned as if each item was cloned individually."
	return unix.Clonefile(sourceDir, destinationDir, 0)
}
