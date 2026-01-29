//go:build linux

package metrics

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/cpu"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/memory"
)

func isCgroupCPU(cpuSource source.CPU) bool {
	_, ok := cpuSource.(*cpu.VersionlessCPU)
	return ok
}

func isCgroupMemory(memorySource source.Memory) bool {
	_, ok := memorySource.(*memory.VersionlessMemory)
	return ok
}
