package build_test

import (
	"testing"

	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/assert"
)

// TestNoUnresolvedDeps ensures that we won't return a task with unresolved dependencies.
func TestNoUnresolvedDeps(t *testing.T) {
	projectDir := testutil.TempDir(t)

	// Create a build with cycle
	b, err := build.New(projectDir, []*api.Task{
		{
			LocalGroupId:   0,
			RequiredGroups: []int64{0},
			Instance:       testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, b.GetNextTask())
}
