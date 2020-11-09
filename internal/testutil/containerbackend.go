package testutil

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"os"
	"testing"
)

func ContainerBackendFromEnv(t *testing.T) containerbackend.ContainerBackend {
	backendName := os.Getenv("CIRRUS_CONTAINER_BACKEND")

	if backendName == "podman" {
		return containerbackend.NewPodman()
	}

	// Default to Docker
	backend, err := containerbackend.NewDocker()
	if err != nil {
		t.Fatal(err)
	}

	return backend
}
