//go:build openbsd || netbsd

package metrics

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"log/slog"
	"time"
)

type Result struct {
	ResourceUtilization *api.ResourceUtilization
}

func (Result) Errors() []error {
	return []error{}
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

type Collector struct{}

func NewCollector(logger *slog.Logger) *Collector {
	return &Collector{}
}

func (collector *Collector) Snapshot() Snapshot {
	return Snapshot{}
}

func (collector *Collector) ResourceUtilizationSnapshot() *api.ResourceUtilization {
	return nil
}

func (collector *Collector) Run(ctx context.Context) chan *Result {
	resultChan := make(chan *Result, 1)

	resultChan <- &Result{}

	return resultChan
}

func Run(ctx context.Context, logger *slog.Logger) chan *Result {
	return NewCollector(logger).Run(ctx)
}
