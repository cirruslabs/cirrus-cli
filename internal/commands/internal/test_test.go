package internal_test

import (
	"bytes"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

// TestTestSimple ensures that a simple test is discovered and ran successfully.
func TestTestSimple(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/test-simple")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"internal", "test", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	require.Nil(t, err)
	assert.Contains(t, buf.String(), "'dir/subdir' succeeded")
}
