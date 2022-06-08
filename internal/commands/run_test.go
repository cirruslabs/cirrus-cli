//go:build !windows
// +build !windows

package commands_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestRun ensures that the run command can handle the simplest possible configuration without problems.
func TestRun(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple"})
	err := command.Execute()

	assert.Nil(t, err)
}

// TestRunTaskFiltering ensures that the task filtering mechanism only runs the task specified by the user.
func TestRunTaskFiltering(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-task-filtering")

	var examples = map[string]struct {
		Pattern         string
		ExpectedStrings []string
	}{
		"first single task": {"first_working", []string{
			"Started 'first_working' task",
			"task first_working (0) succeeded",
		}},
		"second single task": {"Second Working", []string{
			"Started 'Second Working' Task",
			"task Second Working (2) succeeded",
		}},
		"first task case insensitivity": {"FiRsT_WoRkInG", []string{
			"Started 'first_working' task",
			"task first_working (0) succeeded",
		}},
		"second task case insensitivity": {"SECOND WORKING", []string{
			"Started 'Second Working' Task",
			"task Second Working (2) succeeded",
		}},
	}

	for exampleName, example := range examples {
		example := example
		t.Run(exampleName, func(t *testing.T) {
			// Create os.Stderr writer that duplicates it's output to buf
			buf := bytes.NewBufferString("")
			writer := io.MultiWriter(os.Stderr, buf)

			command := commands.NewRootCmd()
			command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", example.Pattern})
			command.SetOut(writer)
			command.SetErr(writer)
			err := command.Execute()

			require.Nil(t, err)
			for _, expectedString := range example.ExpectedStrings {
				require.Contains(t, buf.String(), expectedString)
			}
		})
	}
}

// TestRunTaskDependencyRemoval ensures that dependencies for the task
// selected by the task filtering mechanism are removed properly.
func TestRunTaskDependencyRemoval(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-task-dependency-removal")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "bar"})
	err := command.Execute()

	require.Nil(t, err)
}

// TestRunEnvironmentSet ensures that the user can set environment variables.
func TestRunEnvironmentSet(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-environment")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "-e", "SOMEKEY=good value"})
	err := command.Execute()

	require.Nil(t, err)
}

// TestRunEnvironmentPassThrough ensures that the user can pass-through environment variables
// from the environment where CLI runs.
func TestRunEnvironmentPassThrough(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-environment")

	// Set a variable to be picked up and passed through
	if err := os.Setenv("SOMEKEY", "good value"); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "-e", "SOMEKEY"})
	err := command.Execute()

	require.Nil(t, err)
}

// TestRunEnvironmentPrecedence ensures that user-specified environment variables
// take precedence over variables defined in the configuration.
func TestRunEnvironmentPrecedence(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-environment-precedence")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "-e", "SOMEKEY=good value"})
	err := command.Execute()

	require.Nil(t, err)
}

// TestRunEnvironmentOnlyIf ensures that user-specified environment variables
// are propagated to the configuration parser.
func TestRunEnvironmentOnlyIf(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-environment-only-if")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "-e", "PLEASE_DONT_FAIL=okay"})
	err := command.Execute()

	require.Nil(t, err)
}

// TestRunEnvironmentOnlyIf ensures that base and user environment variables
// are passed to the Starlark execution environment.
func TestRunEnvironmentStarlark(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-environment-starlark")

	// Initialize Git and create a tag for CIRRUS_TAG to be available
	repo, err := git.PlainInit(".", false)
	if err != nil {
		t.Fatal(err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	_, err = worktree.Add(".cirrus.star")
	if err != nil {
		t.Fatal(err)
	}

	commitHash, err := worktree.Commit("0.1.0 release", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Charlie Root",
			Email: "root@localhost",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.CreateTag("v0.1.0", commitHash, nil)
	if err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "-e", "USER_VARIABLE=user variable value"})
	err = command.Execute()

	require.Nil(t, err)
}

// TestRunYAMLAndStarlarkMerged ensures that CLI merges multiple configurations.
func TestRunYAMLAndStarlarkMerged(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-yaml-and-starlark")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.Nil(t, err)
	assert.Contains(t, buf.String(), "'from_yaml' script succeeded")
	assert.Contains(t, buf.String(), "'from_starlark' script succeeded")
}

// TestRunYAMLAndStarlarkHooks ensures that CLI allows Starlark file without main.
func TestRunYAMLAndStarlarkHooks(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-yaml-and-starlark-hooks")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple"})
	err := command.Execute()

	require.Nil(t, err)
}

// TestRunContainerPull ensures that container images are pulled by default.
func TestRunContainerPull(t *testing.T) {
	backend, err := containerbackend.New(containerbackend.BackendAutoType)
	if err != nil {
		t.Fatal(err)
	}

	if err := backend.ImagePull(context.Background(), "debian:latest"); err != nil {
		t.Fatal(err)
	}

	testutil.TempChdirPopulatedWith(t, "testdata/image-pulling-behavior")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "-v", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err = command.Execute()

	require.Nil(t, err)
	assert.Contains(t, strings.ToLower(buf.String()), "pulling image")
	assert.NotContains(t, strings.ToLower(buf.String()), "no such image")
}

// TestRunTaskFilteringByLabel ensures that task filtering logic is label-aware.
func TestRunTaskFilteringByLabel(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-task-filtering-by-label")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "test VERSION:1.14"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "VERSION:1.14")
	assert.NotContains(t, buf.String(), "VERSION:1.15")
}

// TestRunNoCleanup ensures that containers and volumes are kept intact
// after execution ends and --debug-no-cleanup is used.
func TestRunNoCleanup(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-no-cleanup")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "--debug-no-cleanup"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "not cleaning up container")
	assert.Contains(t, buf.String(), "not cleaning up additional container")
	assert.Contains(t, buf.String(), "not cleaning up working volume")

	// The fun ends here since now we have to cleanup containers and volumes ourselves
	backend := testutil.ContainerBackendFromEnv(t)

	containerRegex := regexp.MustCompile("not cleaning up (?:container|additional container) (?P<container_id>[^,]+)")
	volumeRegex := regexp.MustCompile("not cleaning up working volume (?P<volume_id>[^,]+)")

	for _, line := range strings.Split(buf.String(), "\n") {
		matches := containerRegex.FindStringSubmatch(line)
		if matches != nil {
			containerID := matches[containerRegex.SubexpIndex("container_id")]
			if err := backend.ContainerDelete(context.Background(), containerID); err != nil {
				t.Fatal(err)
			}
		}

		matches = volumeRegex.FindStringSubmatch(line)
		if matches != nil {
			volumeID := matches[volumeRegex.SubexpIndex("volume_id")]
			if err := backend.VolumeDelete(context.Background(), volumeID); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestRunNonStandardExtension ensures that we support .cirrus.yaml files.
func TestRunNonStandardExtension(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yaml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple"})
	err := command.Execute()

	assert.Nil(t, err)
}

// TestRunPrebuiltImageTemplate ensures that the user can customize the image name
// that gets built as a part of Dockerfile as CI environment feature[1].
// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
func TestRunPrebuiltImageTemplate(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-prebuilt")

	image := fmt.Sprintf("testing.invalid/%s:latest", uuid.New().String())

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--container-lazy-pull", "-v", "-o simple", "--dockerfile-image-template=" + image})
	err := command.Execute()
	require.NoError(t, err)

	// Make sure the image exists
	backend := testutil.ContainerBackendFromEnv(t)
	err = backend.ImageInspect(context.Background(), image)
	require.NoError(t, err)

	// Cleanup the image
	if err := backend.ImageDelete(context.Background(), image); err != nil {
		t.Fatal(err)
	}
}

func TestAffectedFiles(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-affected-files")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "-v", "-o simple", "--affected-files", "1.txt,2.md"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.Nil(t, err)
	assert.Contains(t, buf.String(), "Debian GNU/Linux")
}

func TestHasStaticEnvironment(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-has-static-environment")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "-v", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()
	require.Nil(t, err)
}

func TestRunGitHubAnnotations(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-github-annotations")

	t.Setenv("GITHUB_ACTIONS", "true")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.NoError(t, err)

	// It's important to check that the workflow command[1] is printed on a separate line
	//
	// [1]: https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions
	lines := strings.Split(buf.String(), "\n")

	assert.Contains(t, lines,
		"::warning file=main.go,line=35,endLine=35,title=use of os.SEEK_START is deprecated::")
	assert.Contains(t, lines,
		"::error file=main_test.go,line=18,endLine=18,title=TestMain() failed!::main_test.go:18: expected a non-nil return")
}
