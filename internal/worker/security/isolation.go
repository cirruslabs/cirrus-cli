package security

import "github.com/IGLOU-EU/go-wildcard"

type IsolationPolicyNone struct {
	// nothing for now
}

type IsolationPolicyContainer struct {
	// nothing for now
}

type IsolationPolicyParallels struct {
	// nothing for now
}

type IsolationPolicyTart struct {
	AllowedImages []string `yaml:"images"`
	ForceSoftnet  bool     `yaml:"force-softnet"`
}

func (tart IsolationPolicyTart) ImageAllowed(name string) bool {
	if len(tart.AllowedImages) == 0 {
		return true
	}

	for _, allowedImage := range tart.AllowedImages {
		if wildcard.MatchSimple(allowedImage, name) {
			return true
		}
	}

	return false
}
