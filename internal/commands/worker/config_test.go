//nolint:testpackage // we need to call the parseConfig(), which is private
package worker

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
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

	require.NotNil(t, config.Security.AllowedIsolations.None)
	require.NotNil(t, config.Security.AllowedIsolations.Container)
	require.Nil(t, config.Security.AllowedIsolations.Tart)
}

func TestRestrictOnlyTartIsolation(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-only-tart-isolation.yml"))
	require.NoError(t, err)

	require.Nil(t, config.Security.AllowedIsolations.None)
	require.Nil(t, config.Security.AllowedIsolations.Container)
	require.NotNil(t, config.Security.AllowedIsolations.Tart)

	require.Empty(t, config.Security.AllowedIsolations.Tart.AllowedImages)
	require.False(t, config.Security.AllowedIsolations.Tart.ForceSoftnet)
}

func TestRestrictOnlySpecificTartImages(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-only-specific-tart-images.yml"))
	require.NoError(t, err)

	require.EqualValues(t, []string{"ghcr.io/cirruslabs/*"},
		config.Security.AllowedIsolations.Tart.AllowedImages)

	const goodImage = "ghcr.io/cirruslabs/macos-ventura-base:latest"
	require.True(t, config.Security.AllowedIsolations.Tart.AllowedImages.ImageAllowed(goodImage))

	badImages := []string{
		"example.com/cirruslabs/macos-ventura-base:latest",
		"example.org/ghcr.io/cirruslabs/whatnot",
	}
	for _, badImage := range badImages {
		assert.False(t, config.Security.AllowedIsolations.Tart.AllowedImages.ImageAllowed(badImage))
	}
}

func TestRestrictForceSoftnet(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "security-force-softnet.yml"))
	require.NoError(t, err)

	require.True(t, config.Security.AllowedIsolations.Tart.ForceSoftnet)
}

func TestStandby(t *testing.T) {
	config, err := parseConfig(filepath.Join("testdata", "standby.yml"))
	require.NoError(t, err)

	// Verify pre-pull configuration exists
	require.NotNil(t, config.TartPrePull)
	require.Equal(t, 3*time.Hour, config.TartPrePull.CheckInterval)

	// Verify pre-pull images
	expectedImages := []string{
		"ghcr.io/cirruslabs/macos-runner:sonoma",
		"ghcr.io/cirruslabs/macos-runner:sequoia",
	}
	require.Equal(t, expectedImages, config.TartPrePull.Images)

	// Verify standby configuration exists
	require.NotNil(t, config.Standby)

	// Verify resources
	require.Equal(t, float64(1), config.Standby.Resources["tart-vms"])

	// Verify isolation configuration
	require.NotNil(t, config.Standby.Isolation)
	require.NotNil(t, config.Standby.Isolation.GetTart())

	tart := config.Standby.Isolation.GetTart()
	require.Equal(t, "ghcr.io/cirruslabs/macos-runner:sonoma", tart.Image)
	require.Equal(t, "admin", tart.User)
	require.Equal(t, "admin", tart.Password)
	require.Equal(t, "1920x1080", tart.Display)
	require.True(t, tart.Softnet)
	require.Equal(t, uint32(4), tart.Cpu)
	require.Equal(t, uint32(16384), tart.Memory)

	// Verify warmup configuration
	require.NotNil(t, config.Standby.Warmup)
	require.Equal(t, "xcrun simctl list || true", config.Standby.Warmup.Script)
	require.Equal(t, uint64(600), config.Standby.Warmup.TimeoutSeconds)
}
