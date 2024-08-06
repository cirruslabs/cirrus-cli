package resourcemodifier_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker/resourcemodifier"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAcquire(t *testing.T) {
	manager := resourcemodifier.NewManager(
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu": 0.5,
			},
		},
	)

	modifier := manager.Acquire(map[string]float64{
		"tart-vms": 1,
		"gpu":      0.5,
	})
	require.NotNil(t, modifier)

	require.Nil(t, manager.Acquire(map[string]float64{
		"tart-vms": 1,
		"gpu":      0.5,
	}))

	modifier.Unlock()

	require.NotNil(t, manager.Acquire(map[string]float64{
		"tart-vms": 1,
		"gpu":      0.5,
	}))
}
