//go:build !(openbsd || netbsd)

package metrics_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second+500*time.Millisecond)
	defer cancel()

	resultChan := metrics.Run(ctx, nil)

	result := <-resultChan

	for i, err := range result.Errors() {
		fmt.Printf("Error #%d: %v\n", i, err)
	}
	require.Empty(t, result.Errors(), "we should never get errors here, but got %d", len(result.Errors()))
	require.Len(t, result.ResourceUtilization.CpuChart, 4)
	require.Len(t, result.ResourceUtilization.MemoryChart, 4)
}

func TestTotals(t *testing.T) {
	ctx := context.Background()

	expectedNumCpusTotal, err := cpu.Counts(true)
	require.NoError(t, err)
	expectedAmountMemory, err := mem.VirtualMemory()
	require.NoError(t, err)

	numCpusTotal, amountMemoryTotal, err := metrics.Totals(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, expectedNumCpusTotal, numCpusTotal)
	assert.EqualValues(t, expectedAmountMemory.Total, amountMemoryTotal)
}
