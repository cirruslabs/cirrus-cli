package github_test

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"syscall"
	"testing"
)

func selfFS() fs.FileSystem {
	return github.New("cirruslabs", "cirrus-cli", "master", "")
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
	assert.True(t, errors.Is(err, syscall.EISDIR))
}
