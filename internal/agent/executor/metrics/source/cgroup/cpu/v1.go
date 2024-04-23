//go:build linux

package cpu

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/parser"
	"os"
	"path/filepath"
)

type V1CPU struct {
	path string
}

func NewV1(path string) (*V1CPU, error) {
	return &V1CPU{
		path: path,
	}, nil
}

func (cpu *V1CPU) CPUUsage() (float64, error) {
	file, err := os.Open(filepath.Join(cpu.path, "cpuacct.usage"))
	if err != nil {
		return 0, err
	}

	result, err := parser.ParseSingleValueFile(file)
	if err != nil {
		return 0, err
	}

	if err := file.Close(); err != nil {
		return 0, err
	}

	return float64(result) / 1_000_000_000, nil
}
