package test_test

import (
	"bytes"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Makes sure all the test cases can be executed successfully.
func TestAll(t *testing.T) {
	fileInfos, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		fileInfo := fileInfo
		t.Run(fileInfo.Name(), func(t *testing.T) {
			if fileInfo.Name() == "update" || fileInfo.Name() == "report" {
				return
			}

			runTestCommandAndGetOutput(t, filepath.Join("testdata", fileInfo.Name()), []string{}, false)
		})
	}
}

// TestSimple ensures that a simple test is discovered and ran successfully.
func TestSimple(t *testing.T) {
	output := runTestCommandAndGetOutput(t, "testdata/simple", []string{}, false)

	adaptedPath := filepath.FromSlash("dir/subdir")
	assert.Contains(t, output, fmt.Sprintf("'%s' succeeded", adaptedPath))
}

func TestReport(t *testing.T) {
	_ = runTestCommandAndGetOutput(t, "testdata/report", []string{"--report", "report-actual.json"}, true)

	expectedReportBytes, err := os.ReadFile("report-expected.json")
	if err != nil {
		t.Fatal(err)
	}

	actualReportBytes, err := os.ReadFile("report-actual.json")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(expectedReportBytes), string(actualReportBytes))
}

func TestUpdate(t *testing.T) {
	_ = runTestCommandAndGetOutput(t, "testdata/update", []string{"--update"}, false)

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	expectedConfigBytes, err := os.ReadFile(filepath.Join(dir, ".cirrus.expected.yml"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `task:
  container:
    image: debian:latest
  script: sleep 5
`, string(expectedConfigBytes))

	logsBytes, err := os.ReadFile(filepath.Join(dir, ".cirrus.expected.log"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `some
log
contents
`, string(logsBytes))
}

// Verify tests succeed and return console output.
func runTestCommandAndGetOutput(t *testing.T, sourceDir string, additionalArgs []string, expectError bool) string {
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

	if err := command.Execute(); expectError {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}

	return buf.String()
}
