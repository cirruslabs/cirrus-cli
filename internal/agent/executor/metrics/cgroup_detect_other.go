//go:build !linux

package metrics

import "github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"

func isCgroupCPU(cpuSource source.CPU) bool {
	return false
}

func isCgroupMemory(memorySource source.Memory) bool {
	return false
}
