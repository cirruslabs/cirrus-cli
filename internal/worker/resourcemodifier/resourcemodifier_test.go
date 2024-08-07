package resourcemodifier_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker/resourcemodifier"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAcquireSimple(t *testing.T) {
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

func TestAcquireIsNotConfusedByValues(t *testing.T) {
	manager := resourcemodifier.NewManager(
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu": 0.5,
			},
			Append: resourcemodifier.Append{
				Run: []string{"--non-existent-argument-unexpected"},
			},
		},
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu": 1,
			},
			Append: resourcemodifier.Append{
				Run: []string{"--non-existent-argument-expected"},
			},
		},
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu": 0.5,
			},
			Append: resourcemodifier.Append{
				Run: []string{"--non-existent-argument-unexpected"},
			},
		},
	)

	modifier := manager.Acquire(map[string]float64{
		"gpu": 1,
	})
	require.NotNil(t, modifier)
	require.Equal(t, []string{"--non-existent-argument-expected"}, modifier.Append.Run)
}

func TestAcquireIsNotConfusedByKeys(t *testing.T) {
	manager := resourcemodifier.NewManager(
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu":                   1,
				"non-existent-resource": 1,
			},
			Append: resourcemodifier.Append{
				Run: []string{"--non-existent-argument-unexpected"},
			},
		},
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu": 1,
			},
			Append: resourcemodifier.Append{
				Run: []string{"--non-existent-argument-expected"},
			},
		},
		&resourcemodifier.Modifier{
			Match: map[string]float64{
				"gpu":                   1,
				"non-existent-resource": 1,
			},
			Append: resourcemodifier.Append{
				Run: []string{"--non-existent-argument-unexpected"},
			},
		},
	)

	modifier := manager.Acquire(map[string]float64{
		"gpu": 1,
	})
	require.NotNil(t, modifier)
	require.Equal(t, []string{"--non-existent-argument-expected"}, modifier.Append.Run)
}
