package volume_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/volume"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestWorkingVolumeSmoke ensures that the working volume gets successfully created and cleaned up.
func TestWorkingVolumeSmoke(t *testing.T) {
	dir := testutil.TempDir(t)

	backend := testutil.ContainerBackendFromEnv(t)

	identifier := uuid.New().String()
	agentVolumeName := fmt.Sprintf("cirrus-agent-volume-%s", identifier)
	workingVolumeName := fmt.Sprintf("cirrus-working-volume-%s", identifier)
	agentVolume, workingVolume, err := volume.CreateWorkingVolume(
		context.Background(),
		backend,
		options.ContainerOptions{},
		agentVolumeName,
		workingVolumeName,
		dir,
		false,
		platform.DefaultAgentVersion,
		platform.Auto(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := agentVolume.Close(backend); err != nil {
		t.Fatal(err)
	}
	if err := workingVolume.Close(backend); err != nil {
		t.Fatal(err)
	}
}

// TestCleanupOnFailure ensures that the not-yet-populated volume gets cleaned up on CreateWorkingVolume() failure.
func TestCleanupOnFailure(t *testing.T) {
	// Create a container backend client
	backend := testutil.ContainerBackendFromEnv(t)

	identifier := uuid.New().String()
	agentVolumeName := fmt.Sprintf("cirrus-agent-volume-%s", identifier)
	workingVolumeName := fmt.Sprintf("cirrus-working-volume-%s", identifier)

	_, _, err := volume.CreateWorkingVolume(
		context.Background(),
		testutil.ContainerBackendFromEnv(t),
		options.ContainerOptions{},
		agentVolumeName,
		workingVolumeName,
		"/non-existent",
		false,
		platform.DefaultAgentVersion,
		platform.Auto(),
	)
	require.Error(t, err)

	err = backend.VolumeInspect(context.Background(), agentVolumeName)
	require.Error(t, err)
	require.True(t, errors.Is(containerbackend.ErrNotFound, err))

	err = backend.VolumeInspect(context.Background(), workingVolumeName)
	require.Error(t, err)
	require.True(t, errors.Is(containerbackend.ErrNotFound, err))
}
