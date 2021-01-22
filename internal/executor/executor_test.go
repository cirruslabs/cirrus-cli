// +build !windows

package executor_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestExecutorEmpty ensures that Executor works fine with an empty task list.
func TestExecutorEmpty(t *testing.T) {
	dir := testutil.TempDir(t)

	e, err := executor.New(dir, []*api.Task{}, executor.WithContainerBackend(testutil.ContainerBackendFromEnv(t)))
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
	}, executor.WithContainerBackend(testutil.ContainerBackendFromEnv(t)))
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

	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

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
	}, executor.WithLogger(logger), executor.WithContainerBackend(testutil.ContainerBackendFromEnv(t)))
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
	}, executor.WithContainerBackend(testutil.ContainerBackendFromEnv(t)))
	if err != nil {
		t.Fatal(err)
	}

	err = e.Run(context.Background())
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, executor.ErrBuildFailed))
}

// TestResourceLimits ensures that the desired CPU and memory limits are enforced for instances.
func TestResourceLimits(t *testing.T) {
	// Skip this test on Podman due to https://github.com/containers/podman/issues/7959
	if _, ok := testutil.ContainerBackendFromEnv(t).(*containerbackend.Podman); ok {
		return
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/resource-limits")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestAdditionalContainers ensures that the services created in the additional containers
// are reachable from the main container.
func TestAdditionalContainers(t *testing.T) {
	// Skip this test on Podman
	if _, ok := testutil.ContainerBackendFromEnv(t).(*containerbackend.Podman); ok {
		return
	}

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
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/docker-pipe-fail-propagation")
	err := testutil.ExecuteWithOptions(t, dir, executor.WithLogger(logger))
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
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/execution-behavior")
	err := testutil.ExecuteWithOptions(t, dir, executor.WithLogger(logger))
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "should_run_because_on_failure")
	assert.Contains(t, buf.String(), "should_run_because_always")
	assert.NotContains(t, buf.String(), "should_not_run_because_on_success")
}

// TestDirtyMode ensures that files created in dirty mode exist on the host.
func TestDirtyMode(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/dirty-mode")

	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	err := testutil.ExecuteWithOptions(t, dir, executor.WithLogger(logger), executor.WithDirtyMode())
	assert.NoError(t, err)

	// Check that the file was created
	_, err = os.Stat(filepath.Join(dir, "file.txt"))
	assert.NoError(t, err)
}

// TestPrebuiltDockerfile ensures that Dockerfile as CI environment[1] feature works properly.
//
// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
func TestPrebuiltDockerfile(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/prebuilt-dockerfile")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestFilesContents ensures that special files (like the one in "dockerfile" field of the container instance)
// actually influence the parsing result.
func TestFilesContents(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/files-contents")

	const (
		debianDockerfile = "FROM debian:latest\n"
		ubuntuDockerfile = "FROM ubuntu:latest\n"
	)

	imageBefore := filesContentsSingleVariation(t, dir, debianDockerfile)
	imageAfter := filesContentsSingleVariation(t, dir, ubuntuDockerfile)
	assert.NotEqual(t, imageBefore, imageAfter)

	imageControl := filesContentsSingleVariation(t, dir, debianDockerfile)
	assert.Equal(t, imageBefore, imageControl)
}

func filesContentsSingleVariation(t *testing.T, dir, dockerfileContents string) string {
	// Re-create the Dockerfile
	dockerfilePath := filepath.Join(dir, "Dockerfile")

	if err := os.Remove(dockerfilePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}

	err := ioutil.WriteFile(dockerfilePath, []byte(dockerfileContents), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Re-parse the configuration
	p := parser.New(parser.WithFileSystem(local.New(dir)))
	result, err := p.ParseFromFile(context.Background(), filepath.Join(dir, ".cirrus.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatal("got parse errors when parsing a known-good configuration")
	}

	// Extract the resulting container instance's image
	for _, task := range result.Tasks {
		inst, err := instance.NewFromProto(task.Instance, []*api.Command{})
		if err != nil {
			continue
		}
		container, ok := inst.(*instance.ContainerInstance)
		if !ok {
			continue
		}

		return container.Image
	}

	t.Fatal("wasn't able to find the container instance in the parsing result")
	return ""
}

// TestPersistentWorker ensures that persistent worker instance is handled properly.
func TestPersistentWorker(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/persistent-worker")
	err := testutil.ExecuteWithOptionsNew(t, dir, executor.WithLogger(logger))
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "'show' script succeeded")
	assert.Contains(t, buf.String(), "'check' script succeeded")
}

// TestCirrusWorkingDir ensures that CIRRUS_WORKING_DIR environment variable is respected.
func TestCirrusWorkingDir(t *testing.T) {
	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/custom-cirrus-working-dir")
	err := testutil.ExecuteWithOptionsNew(t, dir, executor.WithLogger(logger))
	assert.NoError(t, err)
}

// TestDirtyCirrusWorkingDir ensures that CIRRUS_WORKING_DIR environment variable is respected
// when running in --dirty mode.
func TestCirrusWorkingDirDirty(t *testing.T) {
	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/custom-cirrus-working-dir")
	err := testutil.ExecuteWithOptionsNew(t, dir, executor.WithLogger(logger), executor.WithDirtyMode())
	assert.NoError(t, err)
}

// TestLoggingNoExtraNewlines ensures that we don't insert unnecessary
// empty newlines when printing log stream from the agent.
func TestLoggingNoExtraNewlines(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.InfoLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/logging-no-extra-newlines")
	err := testutil.ExecuteWithOptionsNew(t, dir, executor.WithLogger(logger))
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "big gap incoming\n\nwe're still alive\n\x1b[32m'big_gap' script succeeded")
	assert.Contains(t, buf.String(), "no newline in the output\n\x1b[32m'no_newline' script succeeded")
	assert.Contains(t, buf.String(), "double newline in the output\n\n\x1b[32m'double' script succeeded")
}
