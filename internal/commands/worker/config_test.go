//nolint:testpackage // we need to call the parseConfig(), which is private
package worker

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

// TestUnknownFields ensures that we will error out on configuration files
// that have unknown fields.
//
// This is important when using "security:", because even a simple typo
// might result in a non-effectual security configuration.
func TestUnknownFields(t *testing.T) {
	_, err := parseConfig(filepath.Join("testdata", "unknown-fields.yml"))
	require.Error(t, err)
}

func TestRestrictNone(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-none.yml"))
	require.NoError(t, err)

	require.Nil(t, config.Security)
}

func TestRestrictOnlyNoneAndContainerIsolation(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-only-none-and-container-isolation.yml"))
	require.NoError(t, err)

	require.NotNil(t, config.Security.Isolation.None)
	require.NotNil(t, config.Security.Isolation.Container)
	require.Nil(t, config.Security.Isolation.Tart)
}

func TestRestrictOnlyTartIsolation(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-only-tart-isolation.yml"))
	require.NoError(t, err)

	require.Nil(t, config.Security.Isolation.None)
	require.Nil(t, config.Security.Isolation.Container)
	require.NotNil(t, config.Security.Isolation.Tart)

	require.Empty(t, config.Security.Isolation.Tart.AllowedImages)
	require.False(t, config.Security.Isolation.Tart.ForceSoftnet)
}

func TestRestrictOnlySpecificTartImages(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-only-specific-tart-images.yml"))
	require.NoError(t, err)

	require.Equal(t, []string{"ghcr.io/cirruslabs/*"},
		config.Security.Isolation.Tart.AllowedImages)

	const goodImage = "ghcr.io/cirruslabs/macos-ventura-base:latest"
	require.True(t, config.Security.Isolation.Tart.ImageAllowed(goodImage))

	badImages := []string{
		"example.com/cirruslabs/macos-ventura-base:latest",
		"example.org/ghcr.io/cirruslabs/whatnot",
	}
	for _, badImage := range badImages {
		assert.False(t, config.Security.Isolation.Tart.ImageAllowed(badImage))
	}
}

func TestRestrictForceSoftnet(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-force-softnet.yml"))
	require.NoError(t, err)

	require.True(t, config.Security.Isolation.Tart.ForceSoftnet)
}
