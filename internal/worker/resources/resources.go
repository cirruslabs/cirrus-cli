package resources

import (
	"fmt"
	"maps"
)

type Resources map[string]float64

func New() Resources {
	return Resources{}
}

func (r Resources) Add(other Resources) Resources {
	result := maps.Clone(r)

	for key, value := range other {
		result[key] += value
	}

	return result
}

func (r Resources) Sub(other Resources) Resources {
	result := maps.Clone(r)

	for key, value := range other {
		result[key] -= value
	}

	return result
}

func (r Resources) Overlaps(other Resources) bool {
	for otherKey := range other {
		if _, ok := r[otherKey]; ok {
			return true
		}
	}

	return false
}

func (r Resources) CanFit(other Resources) bool {
	for otherKey, otherValue := range other {
		if r[otherKey]-otherValue < 0 {
			return false
		}
	}

	return true
}

func (r Resources) HasPositiveResources() bool {
	for _, value := range r {
		if value > 0 {
			return true
		}
	}

	return false
}

func (r Resources) String() string {
	return fmt.Sprintf("%#v", r)
}
