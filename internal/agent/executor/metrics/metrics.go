//go:build !(openbsd || netbsd)

package metrics

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/cpu"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/memory"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/resolver"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/system"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/dustin/go-humanize"
	gopsutilcpu "github.com/shirou/gopsutil/v3/cpu"
	gopsutilmem "github.com/shirou/gopsutil/v3/mem"
	"github.com/sirupsen/logrus"
	"log/slog"
	"runtime"
	"time"
)

var (
	ErrFailedToQueryTotals = errors.New("failed to query total CPU count/memory amount")
	ErrFailedToQueryCPU    = errors.New("failed to query CPU usage")
	ErrFailedToQueryMemory = errors.New("failed to query memory usage")
)

type Result struct {
	errors              map[string]error
	ResourceUtilization *api.ResourceUtilization
}

func (result Result) Errors() []error {
	var deduplicatedErrors []error

	for _, err := range result.errors {
		deduplicatedErrors = append(deduplicatedErrors, err)
	}

	return deduplicatedErrors
}

func Run(ctx context.Context, logger logrus.FieldLogger) chan *Result {
	resultChan := make(chan *Result, 1)

	var cpuSource source.CPU
	var memorySource source.Memory

	systemSource := system.New()
	cpuSource = systemSource
	memorySource = systemSource

	resolver, err := resolver.New()
	if err != nil {
		if runtime.GOOS == "linux" {
			slog.Warn("cgroup resolver initialization failed, falling back to system-wide metrics collection",
				"err", err)
		}
	} else {
		cpuSrc, err := cpu.NewCPU(resolver)
		if err == nil {
			if logger != nil {
				logger.Infof("CPU metrics are now cgroup-aware")
			}
			cpuSource = cpuSrc
		}

		memorySrc, err := memory.NewMemory(resolver)
		if err == nil {
			if logger != nil {
				logger.Infof("memory metrics are now cgroup-aware")
			}
			memorySource = memorySrc
		}
	}

	go func() {
		result := &Result{
			errors:              map[string]error{},
			ResourceUtilization: &api.ResourceUtilization{},
		}

		// Totals
		numCpusTotal, amountMemoryTotal, totalsErr := Totals(ctx)
		if totalsErr != nil {
			if errors.Is(totalsErr, context.Canceled) || errors.Is(totalsErr, context.DeadlineExceeded) {
				resultChan <- result

				return
			}

			err := fmt.Errorf("%w: %v", ErrFailedToQueryTotals, totalsErr)
			result.errors[err.Error()] = err
		} else {
			result.ResourceUtilization.CpuTotal = float64(numCpusTotal)
			result.ResourceUtilization.MemoryTotal = float64(amountMemoryTotal)
		}

		pollInterval := 1 * time.Second
		startTime := time.Now()

		for {
			cycleStartTime := time.Now()

			// CPU usage
			numCpusUsed, cpuErr := cpuSource.NumCpusUsed(ctx, pollInterval)
			if cpuErr != nil {
				if errors.Is(cpuErr, context.Canceled) || errors.Is(cpuErr, context.DeadlineExceeded) {
					resultChan <- result

					return
				}

				err := fmt.Errorf("%w using %s: %v", ErrFailedToQueryCPU, cpuSource.Name(), cpuErr)
				result.errors[err.Error()] = err
			}

			// Memory usage
			amountMemoryUsed, memoryErr := memorySource.AmountMemoryUsed(ctx)
			if memoryErr != nil {
				if errors.Is(memoryErr, context.Canceled) || errors.Is(memoryErr, context.DeadlineExceeded) {
					resultChan <- result

					return
				}

				err := fmt.Errorf("%w using %s: %v", ErrFailedToQueryMemory, memorySource.Name(), memoryErr)
				result.errors[err.Error()] = err
			}

			if logger != nil {
				logger.Infof("CPUs used: %.2f, CPU usage: %.2f%%, memory used: %s", numCpusUsed, numCpusUsed*100.0,
					humanize.Bytes(uint64(amountMemoryUsed)))
			}

			timeSinceStart := time.Since(startTime)

			if cpuErr == nil {
				result.ResourceUtilization.CpuChart = append(result.ResourceUtilization.CpuChart, &api.ChartPoint{
					SecondsFromStart: uint32(timeSinceStart.Seconds()),
					Value:            numCpusUsed,
				})
			}
			if memoryErr == nil {
				result.ResourceUtilization.MemoryChart = append(result.ResourceUtilization.MemoryChart, &api.ChartPoint{
					SecondsFromStart: uint32(timeSinceStart.Seconds()),
					Value:            amountMemoryUsed,
				})
			}

			// Make sure we wait the whole pollInterval
			timeLeftToWait := pollInterval - time.Since(cycleStartTime)
			select {
			case <-ctx.Done():
				resultChan <- result

				return
			case <-time.After(timeLeftToWait):
				// continue
			}

			// Gradually increase the poll interval to avoid missing data for
			// short-running tasks, but to preserve memory for long-running tasks
			if timeSinceStart > (1 * time.Minute) {
				pollInterval = 10 * time.Second
			}
		}
	}()

	return resultChan
}

func Totals(ctx context.Context) (uint64, uint64, error) {
	perCpuStat, err := gopsutilcpu.TimesWithContext(ctx, true)
	if err != nil {
		return 0, 0, err
	}

	virtualMemoryStat, err := gopsutilmem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}

	return uint64(len(perCpuStat)), virtualMemoryStat.Total, nil
}
