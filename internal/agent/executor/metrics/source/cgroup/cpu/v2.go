//go:build linux

package cpu

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/parser"
	"os"
	"path/filepath"
)

type V2CPU struct {
	path string
}

func NewV2(path string) (*V2CPU, error) {
	return &V2CPU{
		path: path,
	}, nil
}

func (cpu *V2CPU) CPUUsage() (float64, error) {
	file, err := os.Open(filepath.Join(cpu.path, "cpu.stat"))
	if err != nil {
		return 0, err
	}

	kvs, err := parser.ParseKeyValueFile(file)
	if err != nil {
		return 0, err
	}

	const usageUsecFieldName = "usage_usec"
	usageUsec, ok := kvs[usageUsecFieldName]
	if !ok {
		return 0, fmt.Errorf("%w: missing %s field", parser.ErrInvalidFormat, usageUsecFieldName)
	}

	if err := file.Close(); err != nil {
		return 0, err
	}

	return float64(usageUsec) / 1_000_000, nil
}
