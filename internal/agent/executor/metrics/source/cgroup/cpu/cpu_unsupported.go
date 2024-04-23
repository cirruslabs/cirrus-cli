//go:build !linux

package cpu

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/resolver"
)

func NewCPU(resolver resolver.Resolver) (source.CPU, error) {
	return nil, cgroup.ErrUnsupportedPlatform
}
