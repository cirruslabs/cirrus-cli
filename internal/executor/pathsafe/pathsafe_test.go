//go:build darwin || windows

package pathsafe_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/pathsafe"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPathSafeName(t *testing.T) {
	assert.True(t, pathsafe.IsPathSafe("Build"))
	assert.True(t, pathsafe.IsPathSafe("test_25519-42"))
	assert.True(t, pathsafe.IsPathSafe("42"))

	assert.False(t, pathsafe.IsPathSafe(""))
	assert.False(t, pathsafe.IsPathSafe("Tests: Linux"))
	assert.False(t, pathsafe.IsPathSafe("name/that/looks/like/a/path"))
}
