package executor_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
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
	dir := testutil.TempDirPopulatedWith(t, "testdata/resource-limits")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestAdditionalContainers ensures that the services created in the additional containers
// are reachable from the main container.
func TestAdditionalContainers(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/additional-containers")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

func TestCache(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/cache")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// Check that override ENTRYPOINT.
func TestEntrypoint(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/entrypoint")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

func TestGitignore(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/gitignore")

	// Activate .gitignore
	if err := os.Rename(filepath.Join(dir, ".gitignore.inert"), filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(err)
	}

	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestEnvironmentPropagation ensures that the environment variables declared in the
// configuration are propagated to the execution environment.
func TestEnvironmentPropagation(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/environment-propagation")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestEnvironment ensures that environment variables emitted by the CLI are set.
func TestEnvironmentAutomaticVariables(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/environment-automatic-variables")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestDockerPipe ensures that the Docker Pipe commands can communicate through the shared volume.
func TestDockerPipe(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/docker-pipe")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestDockerPipeTermination ensures that the failure in some stage
// of the Docker Pipe is propagated to the next stages.
func TestDockerPipeTermination(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	logger := logrus.New()
	logger.Level = logrus.TraceLevel
	logger.Out = writer

	dir := testutil.TempDirPopulatedWith(t, "testdata/docker-pipe-fail-propagation")
	err := testutil.ExecuteWithLogger(t, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "failing")
	assert.Contains(t, buf.String(), "validate")
	assert.NotContains(t, buf.String(), "never")
}

// TestExecutionBehavior ensures that individual command's execution behavior is respected.
func TestExecutionBehavior(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	logger := logrus.New()
	logger.Level = logrus.TraceLevel
	logger.Out = writer

	dir := testutil.TempDirPopulatedWith(t, "testdata/execution-behavior")
	err := testutil.ExecuteWithLogger(t, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "should_run_because_on_failure")
	assert.Contains(t, buf.String(), "should_run_because_always")
	assert.NotContains(t, buf.String(), "should_not_run_because_on_success")
}
