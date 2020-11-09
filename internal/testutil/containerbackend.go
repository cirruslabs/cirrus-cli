package testutil

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"testing"
)

func ContainerBackendFromEnv(t *testing.T) containerbackend.ContainerBackend {
	backend, err := containerbackend.New(containerbackend.BackendAuto)
	if err != nil {
		t.Fatal(err)
	}

	return backend
}
