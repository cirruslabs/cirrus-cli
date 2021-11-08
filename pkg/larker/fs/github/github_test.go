package github

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"syscall"
	"testing"
)

func selfFS(t *testing.T) fs.FileSystem {
	selfFS, err := New("cirruslabs", "cirrus-cli", "master", "")
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

	testFS := selfFS(t).(*GitHub)
	entries, err := testFS.ReadDir(context.Background(), ".")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, entries, "go.mod", "go.sum")

	cachedContents, ok := testFS.contentsCache.Get("go.mod")
	assert.True(t, ok)
	cachedFileInfo := cachedContents.(*Contents).File
	assert.Nil(t, cachedFileInfo.Content) // partial cache

	_, err = testFS.Get(context.Background(), "go.mod")
	if err != nil {
		t.Fatal(err)
	}

	cachedContents, ok = testFS.contentsCache.Get("go.mod")
	assert.True(t, ok)
	cachedFileInfo = cachedContents.(*Contents).File
	assert.NotNil(t, cachedFileInfo.Content) // verify cache entry got re-populated
}

func TestReadDirNonExistentDirectory(t *testing.T) {
	possiblySkip(t)

	_, err := selfFS(t).ReadDir(context.Background(), "the-directory-that-should-not-exist")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}
