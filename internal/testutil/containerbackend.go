package testutil

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"testing"
)

func ContainerBackendFromEnv(t *testing.T) containerbackend.ContainerBackend {
	// Default to Docker
	backend, err := containerbackend.NewDocker()
	if err != nil {
		t.Fatal(err)
	}

	return backend
}
