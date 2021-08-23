//go:build linux
// +build linux

package executor_test

import (
	"bytes"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

// TestDockerBuilderLinux ensures that Docker Builder instances using Linux platform are supported.
func TestDockerBuilderLinux(t *testing.T) {
	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Create a logger and attach it to writer
	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	dir := testutil.TempDirPopulatedWith(t, "testdata/docker-builder")
	err := testutil.ExecuteWithOptionsNew(t, dir, executor.WithLogger(logger))
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "I am running inside Docker Builder!")
	assert.Contains(t, buf.String(), "'linux' task succeeded")
}
