package executor_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type newlineNormalizingWriter struct {
	writer io.Writer
}

func newNewlineNormalizingWriter(writer io.Writer) *newlineNormalizingWriter {
	return &newlineNormalizingWriter{
		writer: writer,
	}
}

func (nw *newlineNormalizingWriter) Write(p []byte) (n int, err error) {
	return fmt.Fprintln(nw.writer, strings.TrimSpace(string(p)))
}

func TestClone(t *testing.T) {
	dir := t.TempDir()
	env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR":    dir,
		"CIRRUS_REPO_CLONE_URL": "https://github.com/cirrus-modules/cirrus-ci-agent-resolution-strategy-fixture.git",
		"CIRRUS_CHANGE_IN_REPO": "859e9241ad42dae6ac8b9332a7175d8dd96a3f13",
		"CIRRUS_BRANCH":         "main",
	})

	require.True(t, executor.CloneRepository(context.Background(), newNewlineNormalizingWriter(os.Stdout), env))
}

func TestClonePRSameSHA(t *testing.T) {
	dir := t.TempDir()
	env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR":    dir,
		"CIRRUS_REPO_CLONE_URL": "https://github.com/cirrus-modules/cirrus-ci-agent-resolution-strategy-fixture.git",
		"CIRRUS_CHANGE_IN_REPO": "174bc222cad6e1d86fb27111a125a9ce78e7c365",
		"CIRRUS_PR":             "1",
	})

	require.True(t, executor.CloneRepository(context.Background(), newNewlineNormalizingWriter(os.Stdout), env))

	// Read the README.md from the working directory
	readmeBytes, err := os.ReadFile(filepath.Join(dir, "README.md"))
	require.NoError(t, err)

	// Ensure that the working directory DOES NOT CONTAIN a change from `main` that was added
	// after the Outstanding PR was made
	require.NotContains(t, string(readmeBytes), "# `CIRRUS_RESOLUTION_STRATEGY`")
}

func TestClonePRMergeForPRs(t *testing.T) {
	dir := t.TempDir()
	env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR":         dir,
		"CIRRUS_REPO_CLONE_URL":      "https://github.com/cirrus-modules/cirrus-ci-agent-resolution-strategy-fixture.git",
		"CIRRUS_CHANGE_IN_REPO":      "174bc222cad6e1d86fb27111a125a9ce78e7c365",
		"CIRRUS_PR":                  "1",
		"CIRRUS_RESOLUTION_STRATEGY": "MERGE_FOR_PRS",
	})

	require.True(t, executor.CloneRepository(context.Background(), newNewlineNormalizingWriter(os.Stdout), env))

	// Read the README.md from the working directory
	readmeBytes, err := os.ReadFile(filepath.Join(dir, "README.md"))
	require.NoError(t, err)

	// Ensure that the working directory CONTAINS a change from `main` that was added
	// after the Outstanding PR was made
	require.Contains(t, string(readmeBytes), "# `CIRRUS_RESOLUTION_STRATEGY`")
}

func TestCloneActualHEADIsReported(t *testing.T) {
	dir := t.TempDir()

	output := &bytes.Buffer{}

	env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR":         dir,
		"CIRRUS_REPO_CLONE_URL":      "https://github.com/cirruslabs/gradle-example.git",
		"CIRRUS_BRANCH":              "master",
		"CIRRUS_CHANGE_IN_REPO":      "a84fff9ad2177fa3f9a48f535f0067f948e22130",
		"CIRRUS_RESOLUTION_STRATEGY": "SAME_SHA",
	})

	require.True(t, executor.CloneRepository(context.Background(),
		io.MultiWriter(newNewlineNormalizingWriter(os.Stdout), output), env))

	require.Contains(t, output.String(),
		"Checked out a84fff9ad2177fa3f9a48f535f0067f948e22130 on master branch.")
}
