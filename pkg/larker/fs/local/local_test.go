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

func TestStatFile(t *testing.T) {
	// Prepare temporary directory
	dir := testutil.TempDir(t)

	if err := ioutil.WriteFile(filepath.Join(dir, "some-file.txt"), []byte("some-contents"), 0600); err != nil {
		t.Fatal(err)
	}

	stat, err := local.New(dir).Stat(context.Background(), "some-file.txt")
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, stat.IsDir)
}

func TestStatDirectory(t *testing.T) {
	// Prepare temporary directory
	dir := testutil.TempDir(t)

	stat, err := local.New(dir).Stat(context.Background(), ".")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, stat.IsDir)
}

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

func TestPivotNoDotDotBreakout(t *testing.T) {
	lfs := local.New("/chroot")

	pivotedPath, err := lfs.Pivot("../../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "/chroot/etc/passwd", pivotedPath)
}

func TestChdir(t *testing.T) {
	lfs := local.New("/tmp")

	lfs.Chdir("working-directory")

	pivotedPath, err := lfs.Pivot(".")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "/tmp/working-directory", pivotedPath)
}
