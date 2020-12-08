package github_test

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"syscall"
	"testing"
)

func selfFS() fs.FileSystem {
	return github.New("cirruslabs", "cirrus-cli", "master", "")
}

func TestStatFile(t *testing.T) {
	stat, err := selfFS().Stat(context.Background(), "go.mod")
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, stat.IsDir)
}

func TestStatDirectory(t *testing.T) {
	stat, err := selfFS().Stat(context.Background(), ".")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, stat.IsDir)
}

func TestGetFile(t *testing.T) {
	fileBytes, err := selfFS().Get(context.Background(), "go.mod")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, string(fileBytes), "module github.com/cirruslabs/cirrus-cli")
}

func TestGetDirectory(t *testing.T) {
	_, err := selfFS().Get(context.Background(), ".")

	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNormalizedIsADirectory))
}

func TestGetNonExistentFile(t *testing.T) {
	_, err := selfFS().Get(context.Background(), "the-file-that-should-not-exist.txt")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestReadDirFile(t *testing.T) {
	_, err := selfFS().ReadDir(context.Background(), "go.mod")

	require.Error(t, err)
	assert.True(t, errors.Is(err, syscall.ENOTDIR))
}

func TestReadDirDirectory(t *testing.T) {
	entries, err := selfFS().ReadDir(context.Background(), ".")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, entries, "go.mod", "go.sum")
}

func TestReadDirNonExistentDirectory(t *testing.T) {
	_, err := selfFS().ReadDir(context.Background(), "the-directory-that-should-not-exist")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}
