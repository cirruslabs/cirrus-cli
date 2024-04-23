//go:build linux

package memory

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/parser"
	"os"
	"path/filepath"
)

type V2Memory struct {
	path string
}

func NewV2(path string) (*V2Memory, error) {
	return &V2Memory{
		path: path,
	}, nil
}

func (memory *V2Memory) MemoryUsage() (float64, error) {
	file, err := os.Open(filepath.Join(memory.path, "memory.current"))
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

	const inactiveFileFieldName = "inactive_file"
	inactiveFile, ok := kvs[inactiveFileFieldName]
	if !ok {
		return 0, fmt.Errorf("%w: missing %s field", parser.ErrInvalidFormat, inactiveFileFieldName)
	}

	if err := file.Close(); err != nil {
		return 0, err
	}

	return float64(total - inactiveFile), nil
}
