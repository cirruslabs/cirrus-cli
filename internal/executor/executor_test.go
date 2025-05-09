//go:build !windows
// +build !windows

package executor_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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

	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

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
	}, executor.WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// TestExecutorScript ensures that Executor can run a few simple commands.
func TestExecutorScript(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	if _, ok := testutil.ContainerBackendFromEnv(t).(*containerbackend.Podman); ok {
		t.Skip("skipping test on Podman due to https://github.com/containers/podman/issues/7959")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/resource-limits")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestAdditionalContainers ensures that the services created in the additional containers
// are reachable from the main container.
func TestAdditionalContainers(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	if _, ok := testutil.ContainerBackendFromEnv(t).(*containerbackend.Podman); ok {
		t.Skip("skipping test on Podman due to https://github.com/containers/podman/issues/7959")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/additional-containers")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

func TestCache(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/cache")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

func TestCacheOptimisticRestore(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/cache-optimistic-restore")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// Check that override ENTRYPOINT.
func TestEntrypoint(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/entrypoint")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

func TestGitignore(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/environment-propagation")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestEnvironment ensures that environment variables emitted by the CLI are set.
func TestEnvironmentAutomaticVariables(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/environment-automatic-variables")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

func TestEnvironmentVariablesFile(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/environment-variables-file")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestDockerPipe ensures that the Docker Pipe commands can communicate through the shared volume.
func TestDockerPipe(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	dir := testutil.TempDirPopulatedWith(t, "testdata/docker-pipe")
	err := testutil.Execute(t, dir)
	assert.NoError(t, err)
}

// TestDockerPipeTermination ensures that the failure in some stage
// of the Docker Pipe is propagated to the next stages.
func TestDockerPipeTermination(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/docker-pipe-fail-propagation")
	err := testutil.ExecuteWithOptions(t, dir, executor.WithLogger(logger))
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "command failing failed")
	assert.Contains(t, buf.String(), "command validate_first succeeded")
	assert.Contains(t, buf.String(), "command validate_second succeeded")
	assert.Contains(t, buf.String(), "command never_before_first was skipped")
	assert.Contains(t, buf.String(), "command never_after_first was skipped")
	assert.Contains(t, buf.String(), "command never_before_second was skipped")
	assert.Contains(t, buf.String(), "command never_after_second was skipped")
}

// TestExecutionBehavior ensures that individual command's execution behavior is respected.
func TestExecutionBehavior(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/execution-behavior")
	err := testutil.ExecuteWithOptions(t, dir, executor.WithLogger(logger))
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "command should_run_because_on_failure succeeded")
	assert.Contains(t, buf.String(), "command should_run_because_always succeeded")
	assert.Contains(t, buf.String(), "command should_not_run_because_on_success was skipped")
}

// TestDirtyMode ensures that files created in dirty mode exist on the host.
func TestDirtyMode(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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

	err := os.WriteFile(dockerfilePath, []byte(dockerfileContents), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Re-parse the configuration
	p := parser.New(parser.WithFileSystem(local.New(dir)))
	result, err := p.ParseFromFile(context.Background(), filepath.Join(dir, ".cirrus.yml"))
	if err != nil {
		t.Fatal(err)
	}

	// Extract the resulting container instance's image
	for _, task := range result.Tasks {
		inst, err := instance.NewFromProto(task.Instance, []*api.Command{}, "", nil)
		if err != nil {
			continue
		}
		containerInstance, ok := inst.(*container.Instance)
		if !ok {
			continue
		}

		return containerInstance.Image
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

func TestPersistentWorkerContainerIsolationVolumes(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	// Make up a name for the directory that we're going to mount inside of the container
	// (it will be created automatically by the executor)
	dirToBeMounted := filepath.Join(os.TempDir(), "cirrus-cli-volume-dir-"+uuid.New().String())

	// Prepare the configuration that creates a new file in that mounted directory
	config := fmt.Sprintf(`persistent_worker:
  isolation:
    container:
      image: debian:latest
      volumes:
        - %s:/dir-to-be-mounted

unix_task:
  show_script: ls -lah /dir-to-be-mounted
  check_script:
    - touch /dir-to-be-mounted/some-file-name.txt
`, dirToBeMounted)

	dirToBeExecutedFrom := testutil.TempDir(t)

	if err := os.WriteFile(filepath.Join(dirToBeExecutedFrom, ".cirrus.yml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	err := testutil.ExecuteWithOptionsNew(t, dirToBeExecutedFrom, executor.WithLogger(logger))
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "'show' script succeeded")
	assert.Contains(t, buf.String(), "'check' script succeeded")

	_, err = os.Stat(filepath.Join(dirToBeMounted, "some-file-name.txt"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestPersistentWorkerDockerfileAsCIEnvironment(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t,
		"testdata/persistent-worker-dockerfile-as-ci-environment")
	err := testutil.ExecuteWithOptionsNew(t, dir, executor.WithLogger(logger))
	assert.NoError(t, err)
}

func TestPersistentWorkerNoneIsolationGracefulTermination(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	// Pre-retrieve the agent's binary since we're getting time sensitive below
	_, err := agent.RetrieveBinary(context.Background(), platform.DefaultAgentVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dir := testutil.TempDirPopulatedWith(t, "testdata/persistent-worker-graceful-termination")
	_ = testutil.ExecuteWithOptionsNewContext(ctx, t, dir, executor.WithLogger(logger))
	assert.Contains(t, buf.String(), "gracefully terminating agent with PID")
	assert.NotContains(t, buf.String(), "killing agent with PID")
	assert.Regexp(t, "agent with PID [0-9]+ exited normally", buf.String())
}

// TestCirrusWorkingDir ensures that CIRRUS_WORKING_DIR environment variable is respected.
func TestCirrusWorkingDir(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

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

// TestContainerLogs ensures that we receive logs from the agent running inside a container.
func TestContainerLogs(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/container-logs")
	err := testutil.ExecuteWithOptions(t, dir, executor.WithLogger(logger))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Getting initial commands...")
	assert.Contains(t, buf.String(), "Sending heartbeat...")
	assert.Contains(t, buf.String(), "Executing main...")

	// Skip last line check for Podman, for which the final part of the log is sometimes skipped. Exact cause unknown,
	// but might be related to the use of connection flushing[1], which will introduce delay for picking up the next
	// log item from runtime.Log().
	// [1]: https://github.com/containers/podman/blob/v3.0.0/pkg/api/handlers/compat/containers_logs.go#L169-L171
	backend := testutil.ContainerBackendFromEnv(t)
	if _, ok := backend.(*containerbackend.Podman); !ok {
		assert.Regexp(t, "container: [0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}"+
			" Background commands to clean up after: [0-9]+", buf.String())
	}
}

// TestUnsupportedInstancesAreSkipped ensures that we skip unsupported instances instead of failing the build.
func TestUnsupportedInstancesAreSkipped(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	tasks := []*api.Task{
		{
			Name: "canary",
			Commands: []*api.Command{
				{
					Name: "main",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{"true"},
						},
					},
				},
			},
		},
	}

	e, err := executor.New(testutil.TempDir(t), tasks, executor.WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, buf.String(), "'canary' task skipped")
}
