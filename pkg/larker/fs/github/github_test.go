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

func selfFS(t *testing.T) fs.FileSystem {
	selfFS, err := github.New("cirruslabs", "cirrus-cli", "master", "")
	if err != nil {
		t.Fatal(err)
	}

	return selfFS
}

func possiblySkip(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_INTERNAL_NO_GITHUB_API_TESTS"); ok {
		t.SkipNow()
	}
}

func TestStatFile(t *testing.T) {
	possiblySkip(t)

	stat, err := selfFS(t).Stat(context.Background(), "go.mod")
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, stat.IsDir)
}

func TestStatDirectory(t *testing.T) {
	possiblySkip(t)

	stat, err := selfFS(t).Stat(context.Background(), ".")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, stat.IsDir)
}

func TestGetFile(t *testing.T) {
	possiblySkip(t)

	fileBytes, err := selfFS(t).Get(context.Background(), "go.mod")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, string(fileBytes), "module github.com/cirruslabs/cirrus-cli")
}

func TestGetDirectory(t *testing.T) {
	possiblySkip(t)

	_, err := selfFS(t).Get(context.Background(), ".")

	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNormalizedIsADirectory))
}

func TestGetNonExistentFile(t *testing.T) {
	possiblySkip(t)

	_, err := selfFS(t).Get(context.Background(), "the-file-that-should-not-exist.txt")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestReadDirFile(t *testing.T) {
	possiblySkip(t)

	_, err := selfFS(t).ReadDir(context.Background(), "go.mod")

	require.Error(t, err)
	assert.True(t, errors.Is(err, syscall.ENOTDIR))
}

func TestReadDirDirectory(t *testing.T) {
	possiblySkip(t)

	entries, err := selfFS(t).ReadDir(context.Background(), ".")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, entries, "go.mod", "go.sum")
}

func TestReadDirNonExistentDirectory(t *testing.T) {
	possiblySkip(t)

	_, err := selfFS(t).ReadDir(context.Background(), "the-directory-that-should-not-exist")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestStatUsesFileInfosCache(t *testing.T) {
	possiblySkip(t)

	fs := selfFS(t).(*github.GitHub)
	require.EqualValues(t, 0, fs.APICallCount(),
		"GitHub FS should be initialized with zero API call count")

	_, err := fs.ReadDir(context.Background(), ".")
	require.NoError(t, err)
	require.EqualValues(t, 1, fs.APICallCount(),
		"ReadDir() should trigger a real API call")

	fileInfo, err := fs.Stat(context.Background(), "go.mod")
	require.NoError(t, err)
	require.False(t, fileInfo.IsDir)
	require.EqualValues(t, 1, fs.APICallCount(),
		"Stat() calls in the root directory should've triggered no additional API calls")

	fileInfo, err = fs.Stat(context.Background(), "pkg")
	require.NoError(t, err)
	require.True(t, fileInfo.IsDir)
	require.EqualValues(t, 1, fs.APICallCount(),
		"Stat() calls in the root directory should've triggered no additional API calls")
}
