package cpu

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/resolver"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/subsystem"
	gopsutilcpu "github.com/shirou/gopsutil/v3/cpu"
	"math"
	"runtime"
	"time"
)

type VersionlessCPU struct {
	versionedCPU CPUWithUsage
}

func NewCPU(resolver resolver.Resolver) (source.CPU, error) {
	versionlessCPU := &VersionlessCPU{}

	const desiredSubsystem = subsystem.Cpuacct

	v1path, v2path, err := resolver.Resolve(desiredSubsystem)
	if err != nil {
		return nil, err
	}

	if v1path != "" {
		versionlessCPU.versionedCPU, err = NewV1(v1path)
	} else if v2path != "" {
		versionlessCPU.versionedCPU, err = NewV2(v2path)
	} else {
		err = fmt.Errorf("%w for subsystem %s", cgroup.ErrUnconfigured, desiredSubsystem)
	}

	if err != nil {
		return nil, err
	}

	return versionlessCPU, nil
}

func (cpu *VersionlessCPU) NumCpusUsed(ctx context.Context, pollInterval time.Duration) (float64, error) {
	// Before
	times, err := gopsutilcpu.Times(true)
	if err != nil {
		return 0, err
	}
	var systemBefore float64
	for _, time := range times {
		//nolint:staticcheck // continue to use deprecated function for now, see https://github.com/shirou/gopsutil/pull/1325
		systemBefore += time.Total()
	}

	cgroupBefore, err := cpu.versionedCPU.CPUUsage()
	if err != nil {
		return 0, err
	}

	// Sleep
	if err := contextAwareSleep(ctx, pollInterval); err != nil {
		return 0, err
	}

	// After
	times, err = gopsutilcpu.Times(true)
	if err != nil {
		return 0, err
	}
	var systemAfter float64
	for _, time := range times {
		//nolint:staticcheck // continue to use deprecated function for now, see https://github.com/shirou/gopsutil/pull/1325
		systemAfter += time.Total()
	}

	cgroupAfter, err := cpu.versionedCPU.CPUUsage()
	if err != nil {
		return 0, err
	}

	// Calculate number of CPUs used
	cgroupDelta := cgroupAfter - cgroupBefore
	if cgroupDelta < 0 {
		cgroupDelta = 0
	}

	systemDelta := systemAfter - systemBefore
	if systemDelta < 0 {
		systemDelta = 100
	}

	numCpus := len(times)

	return math.Min(100, math.Max(0, cgroupDelta/systemDelta)) * float64(numCpus), nil
}

func (cpu *VersionlessCPU) Name() string {
	return fmt.Sprintf("cgroup CPU resolver on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func contextAwareSleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
