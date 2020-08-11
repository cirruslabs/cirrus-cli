package instance_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"testing"
)

// TestWorkingVolumeSmoke ensures that the working volume gets successfully created and cleaned up.
func TestWorkingVolumeSmoke(t *testing.T) {
	dir := testutil.TempDir(t)

	volume, err := instance.CreateWorkingVolume(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := volume.Close(); err != nil {
		t.Fatal(err)
	}
}
