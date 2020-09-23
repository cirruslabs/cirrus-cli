package instance_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestWorkingVolumeSmoke ensures that the working volume gets successfully created and cleaned up.
func TestWorkingVolumeSmoke(t *testing.T) {
	dir := testutil.TempDir(t)

	desiredVolumeName := fmt.Sprintf("cirrus-working-volume-%s", uuid.New().String())
	volume, err := instance.CreateWorkingVolume(context.Background(), desiredVolumeName, dir, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := volume.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestCleanupOnFailure ensures that the not-yet-populated volume gets cleaned up on CreateWorkingVolume() failure.
func TestCleanupOnFailure(t *testing.T) {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	desiredVolumeName := fmt.Sprintf("cirrus-working-volume-%s", uuid.New().String())
	_, err = instance.CreateWorkingVolume(context.Background(), desiredVolumeName, "/non-existent", false)
	require.Error(t, err)

	_, err = cli.VolumeInspect(context.Background(), desiredVolumeName)
	require.Error(t, err)
	require.True(t, client.IsErrNotFound(err))
}
