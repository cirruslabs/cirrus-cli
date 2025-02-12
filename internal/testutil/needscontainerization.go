package testutil

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"os"
	"testing"
)

// NeedsContainerization skips the test when running on CI, but no container backend configured.
//
// This is needed to prevent failures on platforms without containerization support.
func NeedsContainerization(t *testing.T) {
	t.Helper()

	if _, ok := os.LookupEnv("CI"); !ok {
		// Not running on CI
		return
	}

	if cirrusContainerBackend, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); ok {
		// Running on CI and container backend is configured, but not supported
		if cirrusContainerBackend == containerbackend.BackendTypePodman {
			t.Skip("Podman container backend is not supported, skipping test...")
		}

		// Running on CI and container backend is configured
		return
	}

	// Running on CI and container backend is NOT configured
	t.Skip("running in CI, but no container backend configured, skipping test...")
}
