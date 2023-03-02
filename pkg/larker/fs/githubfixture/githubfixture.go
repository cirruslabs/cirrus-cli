package githubfixture

import (
	"context"
	"errors"
	fspkg "github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"syscall"
	"testing"
)

const (
	URL       = "https://github.com/cirruslabs/cirrus-cli"
	Owner     = "cirruslabs"
	Repo      = "cirrus-cli"
	Reference = "master"
)

func Run(t *testing.T, fs fspkg.FileSystem) {
	ctx := context.Background()

	t.Run("TestStatFile", func(t *testing.T) {
		stat, err := fs.Stat(ctx, "go.mod")
		if err != nil {
			t.Fatal(err)
		}

		assert.False(t, stat.IsDir)
	})

	t.Run("TestStatDirectory", func(t *testing.T) {
		stat, err := fs.Stat(ctx, ".")
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, stat.IsDir)
	})

	t.Run("TestGetFile", func(t *testing.T) {
		fileBytes, err := fs.Get(ctx, "go.mod")
		if err != nil {
			t.Fatal(err)
		}

		assert.Contains(t, string(fileBytes), "module github.com/cirruslabs/cirrus-cli")
	})

	t.Run("TestGetDirectory", func(t *testing.T) {
		_, err := fs.Get(ctx, ".")

		require.Error(t, err)
	})

	t.Run("TestGetNonExistentFile", func(t *testing.T) {
		_, err := fs.Get(ctx, "the-file-that-should-not-exist.txt")

		require.Error(t, err)
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})

	t.Run("TestReadDirFile", func(t *testing.T) {
		_, err := fs.ReadDir(ctx, "go.mod")

		require.Error(t, err)
		assert.True(t, errors.Is(err, syscall.ENOTDIR))
	})

	t.Run("TestReadDirDirectory", func(t *testing.T) {
		entries, err := fs.ReadDir(ctx, ".")
		if err != nil {
			t.Fatal(err)
		}

		assert.Contains(t, entries, "go.mod", "go.sum")
	})

	t.Run("TestReadDirNonExistentDirectory", func(t *testing.T) {
		_, err := fs.ReadDir(ctx, "the-directory-that-should-not-exist")

		require.Error(t, err)
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
}
