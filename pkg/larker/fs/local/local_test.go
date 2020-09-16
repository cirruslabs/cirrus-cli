package local_test

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestGetFile(t *testing.T) {
	// Prepare temporary directory
	dir := testutil.TempDir(t)
	if err := ioutil.WriteFile(filepath.Join(dir, "some-file.txt"), []byte("some-contents"), 0600); err != nil {
		t.Fatal(err)
	}

	fileBytes, err := local.New(dir).Get(context.Background(), "some-file.txt")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, string(fileBytes), "some-contents")
}

func TestGetDirectory(t *testing.T) {
	// Prepare temporary directory
	dir := testutil.TempDir(t)

	_, err := local.New(dir).Get(context.Background(), ".")

	require.Error(t, err)
	assert.True(t, errors.Is(err, syscall.EISDIR))
}

func TestGetNonExistentFile(t *testing.T) {
	// Prepare temporary directory
	dir := testutil.TempDir(t)

	_, err := local.New(dir).Get(context.Background(), "the-file-that-should-not-exist.txt")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}
