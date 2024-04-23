//go:build linux

package memory

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/parser"
	"os"
	"path/filepath"
)

type V1Memory struct {
	path string
}

func NewV1(path string) (*V1Memory, error) {
	return &V1Memory{
		path: path,
	}, nil
}

func (memory *V1Memory) MemoryUsage() (float64, error) {
	file, err := os.Open(filepath.Join(memory.path, "memory.usage_in_bytes"))
	if err != nil {
		return 0, err
	}

	total, err := parser.ParseSingleValueFile(file)
	if err != nil {
		return 0, err
	}

	if err := file.Close(); err != nil {
		return 0, err
	}

	file, err = os.Open(filepath.Join(memory.path, "memory.stat"))
	if err != nil {
		return 0, err
	}

	kvs, err := parser.ParseKeyValueFile(file)
	if err != nil {
		return 0, err
	}

	const totalInactiveFileFieldName = "total_inactive_file"
	totalInactiveFile, ok := kvs[totalInactiveFileFieldName]
	if !ok {
		return 0, fmt.Errorf("%w: missing %s field", parser.ErrInvalidFormat, totalInactiveFileFieldName)
	}

	if err := file.Close(); err != nil {
		return 0, err
	}

	return float64(total - totalInactiveFile), nil
}
