package pwdir

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func StaticTempDirWithDynamicFallback() (string, error) {
	// Prefer static directory for non-Cirrus CI caches efficiency (e.g. ccache)
	staticTempDir := filepath.Join(os.TempDir(), "cirrus-build")
	if err := os.Mkdir(staticTempDir, 0700); err == nil {
		return staticTempDir, nil
	}

	return ioutil.TempDir("", "cirrus-build-")
}
