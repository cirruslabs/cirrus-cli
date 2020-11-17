package instance_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestWorkingVolumeSmoke ensures that the working volume gets successfully created and cleaned up.
func TestWorkingVolumeSmoke(t *testing.T) {
	dir := testutil.TempDir(t)

	backend := testutil.ContainerBackendFromEnv(t)

	desiredVolumeName := fmt.Sprintf("cirrus-working-volume-%s", uuid.New().String())
	volume, err := instance.CreateWorkingVolume(
		context.Background(),
		backend,
		desiredVolumeName,
		dir,
		false,
		instance.DefaultAgentVersion,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := volume.Close(backend); err != nil {
		t.Fatal(err)
	}
}

// TestCleanupOnFailure ensures that the not-yet-populated volume gets cleaned up on CreateWorkingVolume() failure.
func TestCleanupOnFailure(t *testing.T) {
	// Create a container backend client
	backend := testutil.ContainerBackendFromEnv(t)

	desiredVolumeName := fmt.Sprintf("cirrus-working-volume-%s", uuid.New().String())
	_, err := instance.CreateWorkingVolume(
		context.Background(),
		testutil.ContainerBackendFromEnv(t),
		desiredVolumeName,
		"/non-existent",
		false,
		instance.DefaultAgentVersion,
	)
	require.Error(t, err)

	err = backend.VolumeInspect(context.Background(), desiredVolumeName)
	require.Error(t, err)
	require.True(t, errors.Is(containerbackend.ErrNotFound, err))
}
