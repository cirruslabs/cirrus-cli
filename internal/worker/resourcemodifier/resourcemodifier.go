package resourcemodifier

import (
	"sync"
)

type Modifier struct {
	Match  map[string]float64 `yaml:"match"`
	Append Append             `yaml:"append"`

	sync.Mutex
}

type Append struct {
	Run []string `yaml:"run"`
}

func (modifier *Modifier) Matches(other map[string]float64) bool {
	for matchKey, matchValue := range modifier.Match {
		otherValue := other[matchKey]

		if otherValue < matchValue {
			return false
		}
	}

	return true
}

type Manager struct {
	resourceModifiers []*Modifier

	mtx sync.Mutex
}

func NewManager(resourceModifiers ...*Modifier) *Manager {
	return &Manager{
		resourceModifiers: resourceModifiers,
	}
}

func (manager *Manager) Acquire(resources map[string]float64) *Modifier {
	manager.mtx.Lock()
	defer manager.mtx.Unlock()

	for _, resourceModifier := range manager.resourceModifiers {
		if !resourceModifier.Matches(resources) {
			continue
		}

		if resourceModifier.TryLock() {
			return resourceModifier
		}
	}

	return nil
}
