//go:build !linux

package resolver

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup"
)

func New() (Resolver, error) {
	return nil, cgroup.ErrUnsupportedPlatform
}
