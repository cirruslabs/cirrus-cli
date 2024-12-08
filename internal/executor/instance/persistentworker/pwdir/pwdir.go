package pwdir

import (
	"github.com/cirruslabs/cirrus-cli/pkg/privdrop"
	"os"
	"path/filepath"
)

func StaticTempDirWithDynamicFallback() (string, error) {
	// Prefer static directory for non-Cirrus CI caches efficiency (e.g. ccache)
	staticTempDir := filepath.Join(os.TempDir(), "cirrus-build")
	if err := os.Mkdir(staticTempDir, 0700); err == nil {
		// Make sure that static directory belongs to the privilege-dropped
		// user and group, in case privilege dropping was requested
		if chownTo := privdrop.ChownTo; chownTo != nil {
			if err := os.Chown(staticTempDir, chownTo.UID, chownTo.GID); err != nil {
				return "", err
			}
		}

		return staticTempDir, nil
	}

	tempDir, err := os.MkdirTemp("", "cirrus-build-")
	if err != nil {
		return "", err
	}

	// Make sure that the temporary directory belongs to the privilege-dropped
	// user and group, in case privilege dropping was requested
	if chownTo := privdrop.ChownTo; chownTo != nil {
		if err := os.Chown(tempDir, chownTo.UID, chownTo.GID); err != nil {
			return "", err
		}
	}

	return tempDir, nil
}
