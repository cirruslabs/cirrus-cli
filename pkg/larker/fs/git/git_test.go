package git_test

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/git"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"syscall"
	"testing"
)

func fileSystemsToTest(t *testing.T) map[string]fs.FileSystem {
	ghFS, err := github.New("cirruslabs", "cirrus-cli", "master", "")
	if err != nil {
		t.Fatal(err)
	}
	gitFS, err := git.New(context.Background(), "https://github.com/cirruslabs/cirrus-cli", "master")
	if err != nil {
		t.Fatal(err)
	}

	return map[string]fs.FileSystem{"github": ghFS, "git": gitFS}
}

func possiblySkip(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_INTERNAL_NO_GITHUB_API_TESTS"); ok {
		t.SkipNow()
	}
}

func TestStatFile(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			stat, err := fileSystem.Stat(context.Background(), "go.mod")
			if err != nil {
				t.Fatal(err)
			}

			assert.False(t, stat.IsDir)
		})
	}
}

func TestStatDirectory(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			stat, err := fileSystem.Stat(context.Background(), ".")
			if err != nil {
				t.Fatal(err)
			}

			assert.True(t, stat.IsDir)
		})
	}
}

func TestGetFile(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			fileBytes, err := fileSystem.Get(context.Background(), "go.mod")
			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t, string(fileBytes), "module github.com/cirruslabs/cirrus-cli")
		})
	}
}

func TestGetDirectory(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			_, err := fileSystem.Get(context.Background(), ".")

			require.Error(t, err)
			assert.True(t, errors.Is(err, fs.ErrNormalizedIsADirectory))
		})
	}
}

func TestGetNonExistentFile(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			_, err := fileSystem.Get(context.Background(), "the-file-that-should-not-exist.txt")

			require.Error(t, err)
			assert.True(t, errors.Is(err, os.ErrNotExist))
		})
	}
}

func TestReadDirFile(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			_, err := fileSystem.ReadDir(context.Background(), "go.mod")

			require.Error(t, err)
			assert.True(t, errors.Is(err, syscall.ENOTDIR))
		})
	}
}

func TestReadDirDirectory(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			entries, err := fileSystem.ReadDir(context.Background(), ".")
			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t, entries, "go.mod", "go.sum")
		})
	}
}

func TestReadDirNonExistentDirectory(t *testing.T) {
	possiblySkip(t)

	for name, currentFS := range fileSystemsToTest(t) {
		fileSystem := currentFS
		t.Run(name, func(t *testing.T) {
			_, err := fileSystem.ReadDir(context.Background(), "the-directory-that-should-not-exist")

			require.Error(t, err)
			assert.True(t, errors.Is(err, os.ErrNotExist))
		})
	}
}
