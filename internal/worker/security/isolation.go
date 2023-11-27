package security

import (
	"github.com/IGLOU-EU/go-wildcard"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"path/filepath"
	"strings"
)

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
	AllowedImages  AllowedImages       `yaml:"allowed-images"`
	AllowedVolumes []AllowedVolumeTart `yaml:"allowed-volumes"`
	ForceSoftnet   bool                `yaml:"force-softnet"`
}

type AllowedVolumeTart struct {
	Source        string `yaml:"source"`
	ForceReadOnly bool   `yaml:"force-readonly"`
}

type IsolationPolicyVetu struct {
	AllowedImages AllowedImages `yaml:"allowed-images"`
}

func (tart IsolationPolicyTart) VolumeAllowed(volume *api.Isolation_Tart_Volume) bool {
	if len(tart.AllowedVolumes) == 0 {
		return false
	}

	// Clean source file path
	sourceCleaned := filepath.Clean(volume.Source)

	// Preserve separator at the end of the source file path
	if strings.HasSuffix(volume.Source, string(filepath.Separator)) {
		sourceCleaned += string(filepath.Separator)
	}

	for _, allowedVolume := range tart.AllowedVolumes {
		if wildcard.MatchSimple(allowedVolume.Source, sourceCleaned) {
			if allowedVolume.ForceReadOnly {
				return volume.ReadOnly
			}

			return true
		}
	}

	return false
}
