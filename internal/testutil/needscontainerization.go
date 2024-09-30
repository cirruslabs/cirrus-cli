package testutil

import (
	"os"
	"testing"
)

// NeedsContainerization skips the test when running on CI, but no container backend configured.
//
// This is needed to prevent failures on platforms without containerization support.
func NeedsContainerization(t *testing.T) {
	t.Helper()

	_, ci := os.LookupEnv("CI")
	_, cirrusContainerBackend := os.LookupEnv("CIRRUS_CONTAINER_BACKEND")

	if ci && !cirrusContainerBackend {
		t.Skip("running in CI, but no container backend configured, skipping test...")
	}
}
