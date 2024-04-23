package system

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"runtime"
	"time"
)

type System struct{}

func New() *System {
	return &System{}
}

func (system *System) NumCpusUsed(ctx context.Context, pollInterval time.Duration) (float64, error) {
	percentages, err := cpu.PercentWithContext(ctx, pollInterval, true)
	if err != nil {
		return 0, err
	}

	var numCpusUsed float64

	for _, singleCpuUsageInPercents := range percentages {
		numCpusUsed += singleCpuUsageInPercents / 100
	}

	return numCpusUsed, nil
}

func (system *System) AmountMemoryUsed(ctx context.Context) (float64, error) {
	virtualMemoryStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, err
	}

	return float64(virtualMemoryStat.Used), nil
}

func (system *System) Name() string {
	return fmt.Sprintf("gopsutil on %s/%s", runtime.GOOS, runtime.GOARCH)
}
