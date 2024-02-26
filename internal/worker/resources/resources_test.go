package resources_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker/resources"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestResourcesAdd(t *testing.T) {
	targetOriginal := resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 1.0,
	}

	targetNew := targetOriginal.Add(resources.Resources{
		"vetu-vms":     1.0,
		"new-resource": 42.0,
	})

	// Ensure that the original resources map was unchanged
	require.Equal(t, resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 1.0,
	}, targetOriginal)

	// Ensure that the addition works
	require.Equal(t, resources.Resources{
		"tart-vms":     2.0,
		"vetu-vms":     2.0,
		"new-resource": 42.0,
	}, targetNew)
}

func TestResourcesSub(t *testing.T) {
	targetOriginal := resources.Resources{
		"tart-vms":     2.0,
		"vetu-vms":     2.0,
		"old-resource": 42.0,
	}

	targetNew := targetOriginal.Sub(resources.Resources{
		"vetu-vms":     1.0,
		"old-resource": 42.0,
	})

	// Ensure that the original resources map was unchanged
	require.Equal(t, resources.Resources{
		"tart-vms":     2.0,
		"vetu-vms":     2.0,
		"old-resource": 42.0,
	}, targetOriginal)

	// Ensure that the addition works
	require.Equal(t, resources.Resources{
		"tart-vms":     2.0,
		"vetu-vms":     1.0,
		"old-resource": 0.0,
	}, targetNew)
}

func TestResourcesOverlaps(t *testing.T) {
	target := resources.Resources{
		"tart-vms":     2.0,
		"vetu-vms":     2.0,
		"old-resource": 42.0,
	}

	// Ensure that overlapping check works
	// despite the value specified
	require.True(t, target.Overlaps(resources.Resources{
		"tart-vms": 0.1,
	}))
	require.False(t, target.Overlaps(resources.Resources{
		"other-vms": 0.1,
	}))

	// Ensure that overlapping check returns false
	// when there's nothing to overlap
	require.False(t, target.Overlaps(resources.Resources{}))
}

func TestResourcesCanFit(t *testing.T) {
	target := resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 1.0,
	}

	// Ensure that fitting check works
	require.True(t, target.CanFit(resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 0.99,
	}))
	require.True(t, target.CanFit(resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 1.00,
	}))
	require.False(t, target.CanFit(resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 2.0,
	}))

	// Ensure that fitting check returns true
	// when there's nothing to fit
	require.True(t, target.CanFit(resources.Resources{}))
}

func TestResourcesPositiveResources(t *testing.T) {
	require.True(t, resources.Resources{
		"tart-vms": 2.0,
		"vetu-vms": 1.0,
	}.HasPositiveResources())

	require.False(t, resources.Resources{
		"other-vms": -0.5,
	}.HasPositiveResources())
}
