//go:build !windows

package security_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker/security"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsolationPolicyTartVolumeAllowed(t *testing.T) {
	policy := security.IsolationPolicyTart{
		AllowedVolumes: []security.AllowedVolumeTart{
			{Source: "/tmp"},
			{Source: "/var/cache/*"},
		},
	}

	// Non-wildcard checks
	assert.True(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/tmp",
	}))
	assert.False(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/tmp/subdir",
	}))
	assert.False(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/tmp/../../etc/passwd",
	}))

	// Wildcard checks
	assert.True(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/var/cache/",
	}))
	assert.True(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/var/cache/subdir",
	}))
	assert.False(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/var/cache",
	}))
	assert.False(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{
		Source: "/var/cache/../../etc/passwd",
	}))
}

func TestIsolationPolicyTartVolumeAllowedForceReadonly(t *testing.T) {
	policy := security.IsolationPolicyTart{
		AllowedVolumes: []security.AllowedVolumeTart{
			{Source: "/read-write"},
			{Source: "/read-only", ForceReadOnly: true},
		},
	}

	// Required: none, using read-write
	assert.True(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{Source: "/read-write"}))

	// Required: none, using read-only
	assert.True(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{Source: "/read-write", ReadOnly: true}))

	// Required: read-only, using read-write
	assert.False(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{Source: "/read-only"}))

	// Required: read-only, using read-only
	assert.True(t, policy.VolumeAllowed(&api.Isolation_Tart_Volume{Source: "/read-only", ReadOnly: true}))
}
