package executor_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// TestExecutorEmpty ensures that Executor works fine with an empty task list.
func TestExecutorEmpty(t *testing.T) {
	dir := testutil.TempDir(t)

	e, err := executor.New(dir, []*api.Task{})
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// TestExecutorClone ensures that Executor handles clone instruction properly.
func TestExecutorClone(t *testing.T) {
	dir := testutil.TempDir(t)

	// Create a canary file
	const canaryFile = "canary.file"
	file, err := os.Create(filepath.Join(dir, canaryFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	e, err := executor.New(dir, []*api.Task{
		{
			LocalGroupId: 0,
			Name:         "main",
			Commands: []*api.Command{
				{
					Name: "clone",
					Instruction: &api.Command_CloneInstruction{
						CloneInstruction: &api.CloneInstruction{},
					},
				},
				{
					Name: "check",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{fmt.Sprintf("test -e %s", canaryFile)},
						},
					},
				},
			},
			Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// TestExecutorScript ensures that Executor can run a few simple commands.
func TestExecutorScript(t *testing.T) {
	dir := testutil.TempDir(t)

	logger := logrus.New()
	logger.Level = logrus.TraceLevel

	e, err := executor.New(dir, []*api.Task{
		{
			LocalGroupId: 0,
			Name:         "mainTask",
			Commands: []*api.Command{
				{
					Name: "firstCommand",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{
								"date",
							},
						},
					},
				},
				{
					Name: "secondCommand",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{
								"uname -a",
							},
						},
					},
				},
			},
			Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	}, executor.WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// TestExecutorFails ensures that we get an ErrBuildFailed when running
// a build with a deliberately failing command.
func TestExecutorFails(t *testing.T) {
	dir := testutil.TempDir(t)

	e, err := executor.New(dir, []*api.Task{
		{
			LocalGroupId: 0,
			Name:         "mainTask",
			Commands: []*api.Command{
				{
					Name: "failingCommand",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{
								"false",
							},
						},
					},
				},
			},
			Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = e.Run(context.Background())
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, executor.ErrBuildFailed))
}

// TestResourceLimits ensures that the desired CPU and memory limits are enforced for instances.
func TestResourceLimits(t *testing.T) {
	testutil.Execute(t, "testdata/resource-limits")
}

// TestAdditionalContainers ensures that the services created in the additional containers
// are reachable from the main container.
func TestAdditionalContainers(t *testing.T) {
	testutil.Execute(t, "testdata/additional-containers")
}

func TestCache(t *testing.T) {
	testutil.Execute(t, "testdata/cache")
}
