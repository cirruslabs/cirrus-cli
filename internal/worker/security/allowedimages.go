package security

import "github.com/IGLOU-EU/go-wildcard"

type AllowedImages []string

func (allowedImages AllowedImages) ImageAllowed(name string) bool {
	if len(allowedImages) == 0 {
		return true
	}

	for _, allowedImage := range allowedImages {
		if wildcard.MatchSimple(allowedImage, name) {
			return true
		}
	}

	return false
}
