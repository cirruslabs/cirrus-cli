package test_test

import (
	"bytes"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Makes sure all the test cases can be executed successfully.
func TestAll(t *testing.T) {
	fileInfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		fileInfo := fileInfo
		t.Run(fileInfo.Name(), func(t *testing.T) {
			runTestCommandAndGetOutput(t, filepath.Join("testdata", fileInfo.Name()), []string{})
		})
	}
}

// TestSimple ensures that a simple test is discovered and ran successfully.
func TestSimple(t *testing.T) {
	output := runTestCommandAndGetOutput(t, "testdata/simple", []string{})

	adaptedPath := filepath.FromSlash("dir/subdir")
	assert.Contains(t, output, fmt.Sprintf("'%s' succeeded", adaptedPath))
}

func TestUpdate(t *testing.T) {
	_ = runTestCommandAndGetOutput(t, "testdata/update", []string{"--update"})

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	expectedConfigBytes, err := ioutil.ReadFile(filepath.Join(dir, ".cirrus.expected.yml"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `task:
  container:
    image: debian:latest
  script: sleep 5
`, string(expectedConfigBytes))

	logsBytes, err := ioutil.ReadFile(filepath.Join(dir, ".cirrus.expected.log"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `some
log
contents
`, string(logsBytes))
}

// Verify tests succeed and return console output.
func runTestCommandAndGetOutput(t *testing.T, sourceDir string, additionalArgs []string) string {
	testutil.TempChdirPopulatedWith(t, sourceDir)

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()

	args := []string{"internal", "test", "-o simple"}
	args = append(args, additionalArgs...)
	command.SetArgs(args)

	command.SetOut(writer)
	command.SetErr(writer)

	err := command.Execute()

	require.Nil(t, err)

	return buf.String()
}
