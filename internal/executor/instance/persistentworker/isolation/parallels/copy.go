//go:build !darwin

package parallels

import "github.com/otiai10/copy"

func CopyDir(sourceDir string, destinationDir string) error {
	return copy.Copy(sourceDir, destinationDir)
}
