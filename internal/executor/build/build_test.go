package build_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAuth ensures that the authentication works as intended.
func TestAuth(t *testing.T) {
	projectDir := testutil.TempDir(t)

	b, err := build.New(projectDir, []*api.Task{
		{
			LocalGroupId: 0,
			Instance:     testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	validSecret := uuid.New().String()
	invalidSecret := uuid.New().String()

	t.Run("valid credentials", func(t *testing.T) {
		_, err = b.GetTaskFromIdentification(&api.TaskIdentification{
			Secret: validSecret,
			TaskId: 0,
		}, validSecret)
		assert.Nil(t, err)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		_, err = b.GetTaskFromIdentification(&api.TaskIdentification{
			Secret: invalidSecret,
			TaskId: 0,
		}, validSecret)
		assert.NotNil(t, err)
	})
}

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
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, b.GetNextTask())
}
