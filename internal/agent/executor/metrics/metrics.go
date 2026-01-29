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
	"log/slog"
	"runtime"
	"sync"
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

type Snapshot struct {
	Timestamp   time.Time
	CPUUsed     float64
	MemoryUsed  float64
	CPUTotal    float64
	MemoryTotal float64
	CPUError    error
	MemoryError error
	TotalsError error
}

type Collector struct {
	cpuSource    source.CPU
	memorySource source.Memory
	logger       *slog.Logger
	mu           sync.RWMutex
	snapshot     Snapshot
	utilization  *api.ResourceUtilization
}

func NewCollector(logger *slog.Logger) *Collector {
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
				logger.Info("CPU metrics are now cgroup-aware")
			}
			cpuSource = cpuSrc
		}

		memorySrc, err := memory.NewMemory(resolver)
		if err == nil {
			if logger != nil {
				logger.Info("memory metrics are now cgroup-aware")
			}
			memorySource = memorySrc
		}
	}

	return &Collector{
		cpuSource:    cpuSource,
		memorySource: memorySource,
		logger:       logger,
		utilization:  &api.ResourceUtilization{},
	}
}

func (collector *Collector) Snapshot() Snapshot {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	return collector.snapshot
}

func (collector *Collector) ResourceUtilizationSnapshot() *api.ResourceUtilization {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	if collector.utilization == nil {
		return nil
	}

	snapshot := &api.ResourceUtilization{
		CpuTotal:    collector.utilization.CpuTotal,
		MemoryTotal: collector.utilization.MemoryTotal,
	}

	if len(collector.utilization.CpuChart) > 0 {
		snapshot.CpuChart = make([]*api.ChartPoint, len(collector.utilization.CpuChart))
		for i, point := range collector.utilization.CpuChart {
			if point == nil {
				continue
			}
			value := *point
			snapshot.CpuChart[i] = &value
		}
	}

	if len(collector.utilization.MemoryChart) > 0 {
		snapshot.MemoryChart = make([]*api.ChartPoint, len(collector.utilization.MemoryChart))
		for i, point := range collector.utilization.MemoryChart {
			if point == nil {
				continue
			}
			value := *point
			snapshot.MemoryChart[i] = &value
		}
	}

	return snapshot
}

func (collector *Collector) updateTotals(numCpusTotal uint64, amountMemoryTotal uint64, totalsErr error) {
	collector.mu.Lock()
	snapshot := collector.snapshot
	snapshot.TotalsError = totalsErr
	if totalsErr == nil {
		snapshot.CPUTotal = float64(numCpusTotal)
		snapshot.MemoryTotal = float64(amountMemoryTotal)
		if collector.utilization != nil {
			collector.utilization.CpuTotal = float64(numCpusTotal)
			collector.utilization.MemoryTotal = float64(amountMemoryTotal)
		}
	}
	collector.snapshot = snapshot
	collector.mu.Unlock()
}

func (collector *Collector) updateUsage(timeSinceStart time.Duration, numCpusUsed float64, cpuErr error, amountMemoryUsed float64, memoryErr error) {
	collector.mu.Lock()
	snapshot := collector.snapshot
	snapshot.Timestamp = time.Now()
	snapshot.CPUError = cpuErr
	snapshot.MemoryError = memoryErr
	if cpuErr == nil {
		snapshot.CPUUsed = numCpusUsed
	}
	if memoryErr == nil {
		snapshot.MemoryUsed = amountMemoryUsed
	}
	collector.snapshot = snapshot
	if collector.utilization != nil {
		if cpuErr == nil {
			collector.utilization.CpuChart = append(collector.utilization.CpuChart, &api.ChartPoint{
				SecondsFromStart: uint32(timeSinceStart.Seconds()),
				Value:            numCpusUsed,
			})
		}
		if memoryErr == nil {
			collector.utilization.MemoryChart = append(collector.utilization.MemoryChart, &api.ChartPoint{
				SecondsFromStart: uint32(timeSinceStart.Seconds()),
				Value:            amountMemoryUsed,
			})
		}
	}
	collector.mu.Unlock()
}

func (collector *Collector) Run(ctx context.Context) chan *Result {
	resultChan := make(chan *Result, 1)

	go func() {
		result := &Result{
			errors:              map[string]error{},
			ResourceUtilization: collector.utilization,
		}

		// Totals
		numCpusTotal, amountMemoryTotal, totalsErr := Totals(ctx)
		collector.updateTotals(numCpusTotal, amountMemoryTotal, totalsErr)
		if totalsErr != nil {
			if errors.Is(totalsErr, context.Canceled) || errors.Is(totalsErr, context.DeadlineExceeded) {
				resultChan <- result

				return
			}

			err := fmt.Errorf("%w: %v", ErrFailedToQueryTotals, totalsErr)
			result.errors[err.Error()] = err
		}

		pollInterval := 1 * time.Second
		startTime := time.Now()

		for {
			cycleStartTime := time.Now()

			// CPU usage
			numCpusUsed, cpuErr := collector.cpuSource.NumCpusUsed(ctx, pollInterval)
			if cpuErr != nil {
				if errors.Is(cpuErr, context.Canceled) || errors.Is(cpuErr, context.DeadlineExceeded) {
					resultChan <- result

					return
				}

				err := fmt.Errorf("%w using %s: %v", ErrFailedToQueryCPU, collector.cpuSource.Name(), cpuErr)
				result.errors[err.Error()] = err
			}

			// Memory usage
			amountMemoryUsed, memoryErr := collector.memorySource.AmountMemoryUsed(ctx)
			if memoryErr != nil {
				if errors.Is(memoryErr, context.Canceled) || errors.Is(memoryErr, context.DeadlineExceeded) {
					resultChan <- result

					return
				}

				err := fmt.Errorf("%w using %s: %v", ErrFailedToQueryMemory, collector.memorySource.Name(), memoryErr)
				result.errors[err.Error()] = err
			}

			if collector.logger != nil {
				collector.logger.Info("Resource usage",
					"cpus_used", numCpusUsed,
					"cpu_usage_percent", numCpusUsed*100.0,
					"memory_used", humanize.Bytes(uint64(amountMemoryUsed)))
			}

			timeSinceStart := time.Since(startTime)
			collector.updateUsage(timeSinceStart, numCpusUsed, cpuErr, amountMemoryUsed, memoryErr)

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

func Run(ctx context.Context, logger *slog.Logger) chan *Result {
	return NewCollector(logger).Run(ctx)
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
