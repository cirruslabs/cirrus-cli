//go:build !linux

package memory

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/resolver"
)

func NewMemory(resolver resolver.Resolver) (source.Memory, error) {
	return nil, cgroup.ErrUnsupportedPlatform
}
