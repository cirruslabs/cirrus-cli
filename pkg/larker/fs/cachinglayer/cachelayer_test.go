package cachinglayer_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/cachinglayer"
	"github.com/stretchr/testify/require"
	"path"
	"testing"
)

const fileContentsFixture = "some file contents"

var directoryContentFixture = []string{"some file.txt", "other file.md"}

type canaryFilesystem struct {
	statCalls    map[string]int
	getCalls     map[string]int
	readDirCalls map[string]int
}

func newCanaryFileSystem() *canaryFilesystem {
	return &canaryFilesystem{
		statCalls:    make(map[string]int),
		getCalls:     make(map[string]int),
		readDirCalls: make(map[string]int),
	}
}

func (cfs *canaryFilesystem) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	cfs.statCalls[path]++

	return &fs.FileInfo{
		IsDir: true,
	}, nil
}

func (cfs *canaryFilesystem) Get(ctx context.Context, path string) ([]byte, error) {
	cfs.getCalls[path]++

	return []byte(fileContentsFixture), nil
}

func (cfs *canaryFilesystem) ReadDir(ctx context.Context, path string) ([]string, error) {
	cfs.readDirCalls[path]++

	return directoryContentFixture, nil
}

func (cfs *canaryFilesystem) Join(elem ...string) string {
	return path.Join(elem...)
}

func (cfs *canaryFilesystem) StatCount(path string) int    { return cfs.statCalls[path] }
func (cfs *canaryFilesystem) GetCount(path string) int     { return cfs.getCalls[path] }
func (cfs *canaryFilesystem) ReadDirCount(path string) int { return cfs.readDirCalls[path] }

func TestCaching(t *testing.T) {
	ctx := context.Background()

	canaryFS := newCanaryFileSystem()

	wrappedFS, err := cachinglayer.Wrap(canaryFS)
	if err != nil {
		t.Fatal(err)
	}

	// Test Stat()
	const statFile = "1.txt"

	for i := 0; i < 2; i++ {
		fileInfo, err := wrappedFS.Stat(ctx, statFile)
		require.NoError(t, err)
		require.True(t, fileInfo.IsDir)
		require.Equal(t, 1, canaryFS.StatCount(statFile))
	}

	// Test Get()
	const getFile = "2.txt"

	for i := 0; i < 2; i++ {
		fileBytes, err := wrappedFS.Get(ctx, getFile)
		require.NoError(t, err)
		require.Equal(t, fileContentsFixture, string(fileBytes))
		require.Equal(t, 1, canaryFS.GetCount(getFile))
	}

	// Test ReadDir()
	const readDirFile = "dir"

	for i := 0; i < 2; i++ {
		directoryContent, err := wrappedFS.ReadDir(ctx, readDirFile)
		require.NoError(t, err)
		require.Equal(t, directoryContentFixture, directoryContent)
		require.Equal(t, 1, canaryFS.ReadDirCount(readDirFile))
	}
}
