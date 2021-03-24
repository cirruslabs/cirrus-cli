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

// TestSimple ensures that a simple test is discovered and ran successfully.
func TestSimple(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/simple")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"internal", "test", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.Nil(t, err)

	adaptedPath := filepath.FromSlash("dir/subdir")
	assert.Contains(t, buf.String(), fmt.Sprintf("'%s' succeeded", adaptedPath))
}

func TestDryRunAll(t *testing.T) {
	fileInfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		fileInfo := fileInfo
		t.Run(fileInfo.Name(), func(t *testing.T) {
			testutil.TempChdirPopulatedWith(t, filepath.Join("testdata", fileInfo.Name()))

			command := commands.NewRootCmd()
			command.SetArgs([]string{"internal", "test", "-o simple"})
			err := command.Execute()

			require.Nil(t, err)
		})
	}
}
