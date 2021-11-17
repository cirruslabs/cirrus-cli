package github_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestStatUsesFileInfosCache(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_INTERNAL_NO_GITHUB_API_TESTS"); ok {
		t.SkipNow()
	}

	fileSystem, err := github.New("cirruslabs", "cirrus-cli", "master", "")
	if err != nil {
		t.Fatal(err)
	}
	require.EqualValues(t, 0, fileSystem.APICallCount(),
		"GitHub FS should be initialized with zero API call count")

	_, err = fileSystem.ReadDir(context.Background(), ".")
	require.NoError(t, err)
	require.EqualValues(t, 1, fileSystem.APICallCount(),
		"ReadDir() should trigger a real API call")

	fileInfo, err := fileSystem.Stat(context.Background(), "go.mod")
	require.NoError(t, err)
	require.False(t, fileInfo.IsDir)
	require.EqualValues(t, 1, fileSystem.APICallCount(),
		"Stat() calls in the root directory should've triggered no additional API calls")

	fileInfo, err = fileSystem.Stat(context.Background(), "pkg")
	require.NoError(t, err)
	require.True(t, fileInfo.IsDir)
	require.EqualValues(t, 1, fileSystem.APICallCount(),
		"Stat() calls in the root directory should've triggered no additional API calls")
}
