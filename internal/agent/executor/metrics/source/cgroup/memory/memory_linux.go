package memory

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/resolver"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/subsystem"
	"runtime"
)

type VersionlessMemory struct {
	versionedMemory MemoryWithUsage
}

func NewMemory(resolver resolver.Resolver) (source.Memory, error) {
	versionlessMemory := &VersionlessMemory{}

	const desiredSubsystem = subsystem.Memory

	v1path, v2path, err := resolver.Resolve(desiredSubsystem)
	if err != nil {
		return nil, err
	}

	if v1path != "" {
		versionlessMemory.versionedMemory, err = NewV1(v1path)
	} else if v2path != "" {
		versionlessMemory.versionedMemory, err = NewV2(v2path)
	} else {
		err = fmt.Errorf("%w for subsystem %s", cgroup.ErrUnconfigured, desiredSubsystem)
	}

	if err != nil {
		return nil, err
	}

	return versionlessMemory, nil
}

func (cpu *VersionlessMemory) Name() string {
	return fmt.Sprintf("cgroup memory resolver on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func (memory *VersionlessMemory) AmountMemoryUsed(ctx context.Context) (float64, error) {
	return memory.versionedMemory.MemoryUsage()
}
