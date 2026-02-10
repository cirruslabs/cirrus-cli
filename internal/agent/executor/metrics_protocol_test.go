package executor

import (
	"strings"
	"testing"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestFormatGithubActionsNoticeWithChart(t *testing.T) {
	snapshot := metrics.Snapshot{}
	utilization := &api.ResourceUtilization{
		CpuTotal:    4,
		MemoryTotal: 4_000_000_000,
	}
	for i := range 20 {
		utilization.CpuChart = append(utilization.CpuChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            float64(i+1) * 0.1,
		})
		utilization.MemoryChart = append(utilization.MemoryChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            float64(i+1) * 100_000_000,
		})
	}

	notice := formatGithubActionsNotice(snapshot, utilization)

	lines := strings.Split(notice, "\n")
	require.Len(t, lines, 108)
	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 2.00 cores (50.00% of 4.00) at 19s%0APeak memory utilization: 2.0 GB (50.00% of 4.0 GB) at 19s", lines[0])
	require.Equal(t, "Resource utilization charts (asciigraph)", lines[1])
	require.Equal(t, "Time   0s -> 19s", lines[2])
	require.Equal(t, "CPU utilization (% of total, peak 50.00%)", lines[3])
	require.Equal(t, "Memory utilization (% of total, peak 50.00%)", lines[55])
	require.Equal(t, "Legend: y-axis is utilization percent", lines[107])
}

func TestFormatGithubActionsNoticeWithoutChart(t *testing.T) {
	snapshot := metrics.Snapshot{
		CPUUsed:    1.5,
		MemoryUsed: 1_000_000_000,
		Timestamp:  time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
	}

	notice := formatGithubActionsNotice(snapshot, nil)

	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 1.50 cores%0APeak memory utilization: 1.0 GB", notice)
}

func TestAcceptsGithubActions(t *testing.T) {
	require.True(t, acceptsGithubActions("text/vnd.github-actions"))
	require.True(t, acceptsGithubActions("text/plain, text/vnd.github-actions"))
	require.True(t, acceptsGithubActions("text/vnd.github-actions; q=1.0"))
	require.False(t, acceptsGithubActions("text/vnd.github-actions; q=0"))
	require.False(t, acceptsGithubActions("application/json"))
	require.False(t, acceptsGithubActions(""))
}

func TestAcceptsJSON(t *testing.T) {
	require.True(t, acceptsJSON("application/json"))
	require.True(t, acceptsJSON("application/vnd.github+json; charset=utf-8"))
	require.False(t, acceptsJSON("application/json; q=0"))
	require.False(t, acceptsJSON(""))
}

func TestFormatGithubActionsNoticeWithWarning(t *testing.T) {
	utilization := &api.ResourceUtilization{
		CpuTotal:    4,
		MemoryTotal: 4_000_000_000,
		CpuChart: []*api.ChartPoint{
			{SecondsFromStart: 2, Value: 1.5},
		},
		MemoryChart: []*api.ChartPoint{
			{SecondsFromStart: 4, Value: 1_000_000_000},
		},
	}

	notice := formatGithubActionsNotice(metrics.Snapshot{}, utilization)

	lines := strings.Split(notice, "\n")
	require.Len(t, lines, 2)
	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 1.50 cores (37.50% of 4.00) at 2s%0APeak memory utilization: 1.0 GB (25.00% of 4.0 GB) at 4s", lines[0])
	require.Equal(t, "::warning title=Resource Utilization::Peak CPU and memory utilization are below 50% of available resources; it might be worth using a different resource class if possible.", lines[1])
	require.NotContains(t, notice, "Resource utilization charts (asciigraph")
}

func TestFormatGithubActionsNoticeWithoutWarningForCgroupTotals(t *testing.T) {
	utilization := &api.ResourceUtilization{
		CpuTotal:    4,
		MemoryTotal: 4_000_000_000,
		CpuChart: []*api.ChartPoint{
			{SecondsFromStart: 2, Value: 1.5},
		},
		MemoryChart: []*api.ChartPoint{
			{SecondsFromStart: 4, Value: 1_000_000_000},
		},
	}

	snapshot := metrics.Snapshot{
		CPUIsCgroup:    true,
		MemoryIsCgroup: true,
	}

	notice := formatGithubActionsNotice(snapshot, utilization)

	lines := strings.Split(notice, "\n")
	require.Len(t, lines, 1)
	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 1.50 cores (37.50% of 4.00) at 2s%0APeak memory utilization: 1.0 GB (25.00% of 4.0 GB) at 4s", lines[0])
	require.NotContains(t, notice, "Resource utilization charts (asciigraph")
}

func TestFormatGithubActionsNoticeChartWidthCap(t *testing.T) {
	utilization := &api.ResourceUtilization{
		CpuTotal:    8,
		MemoryTotal: 8_000_000_000,
	}

	for i := range 500 {
		utilization.CpuChart = append(utilization.CpuChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            float64((i%8)+1) * 0.75,
		})
		utilization.MemoryChart = append(utilization.MemoryChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            float64((i%8)+1) * 500_000_000,
		})
	}

	chart := formatGithubActionsASCIIChart(utilization, utilization.CpuTotal, utilization.MemoryTotal, githubActionsChartMaxWidth)
	lines := strings.Split(chart, "\n")
	require.Len(t, lines, 107)
	require.Equal(t, "Resource utilization charts (asciigraph)", lines[0])
	require.Equal(t, "Time   0s -> 499s", lines[1])
	require.Equal(t, "CPU utilization (% of total, peak 75.00%)", lines[2])
	require.Equal(t, "Memory utilization (% of total, peak 50.00%)", lines[54])
	require.Equal(t, "Legend: y-axis is utilization percent", lines[106])
}

func TestFormatGithubActionsASCIIChartFullVisualization(t *testing.T) {
	utilization := &api.ResourceUtilization{
		CpuTotal:    16,
		MemoryTotal: 16_000_000_000,
	}

	for i := range 220 {
		cpuPhase := float64(i % 60)
		if cpuPhase > 30 {
			cpuPhase = 60 - cpuPhase
		}
		memPhase := float64((i + 20) % 80)
		if memPhase > 40 {
			memPhase = 80 - memPhase
		}

		utilization.CpuChart = append(utilization.CpuChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            2 + cpuPhase*0.45,
		})
		utilization.MemoryChart = append(utilization.MemoryChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            1_000_000_000 + memPhase*300_000_000,
		})
	}

	chart := formatGithubActionsASCIIChart(utilization, utilization.CpuTotal, utilization.MemoryTotal, githubActionsChartMaxWidth)
	lines := strings.Split(chart, "\n")

	require.Len(t, lines, 107)
	require.Equal(t, "Resource utilization charts (asciigraph)", lines[0])
	require.Equal(t, "Time   0s -> 219s", lines[1])
	require.Equal(t, "CPU utilization (% of total, peak 96.88%)", lines[2])
	require.Equal(t, "Memory utilization (% of total, peak 81.25%)", lines[54])
	require.Equal(t, "Legend: y-axis is utilization percent", lines[106])

	t.Logf("\n%s", chart)
}

func TestFormatGithubActionsNoticeWithoutChartForShortRuns(t *testing.T) {
	utilization := &api.ResourceUtilization{
		CpuTotal:    4,
		MemoryTotal: 4_000_000_000,
	}
	for i := range 14 {
		utilization.CpuChart = append(utilization.CpuChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            3.0,
		})
		utilization.MemoryChart = append(utilization.MemoryChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            3_000_000_000,
		})
	}

	notice := formatGithubActionsNotice(metrics.Snapshot{}, utilization)

	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 3.00 cores (75.00% of 4.00) at 0s%0APeak memory utilization: 3.0 GB (75.00% of 4.0 GB) at 0s", notice)
	require.NotContains(t, notice, "Resource utilization charts (asciigraph")
}

func TestFormatGithubActionsASCIIChartFullMultilineText(t *testing.T) {
	utilization := &api.ResourceUtilization{
		CpuTotal:    4,
		MemoryTotal: 4_000_000_000,
	}
	for i := range 15 {
		utilization.CpuChart = append(utilization.CpuChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            2.0,
		})
		utilization.MemoryChart = append(utilization.MemoryChart, &api.ChartPoint{
			SecondsFromStart: uint32(i),
			Value:            1_000_000_000,
		})
	}

	chart := formatGithubActionsASCIIChart(utilization, utilization.CpuTotal, utilization.MemoryTotal, githubActionsChartMaxWidth)
	expected := `Resource utilization charts (asciigraph)
Time   0s -> 14s
CPU utilization (% of total, peak 50.00%)
 100 ┤
  98 ┤
  96 ┤
  94 ┤
  92 ┤
  90 ┤
  88 ┤
  86 ┤
  84 ┤
  82 ┤
  80 ┤
  78 ┤
  76 ┤
  74 ┤
  72 ┤
  70 ┤
  68 ┤
  66 ┤
  64 ┤
  62 ┤
  60 ┤
  58 ┤
  56 ┤
  54 ┤
  52 ┤
  50 ┼───────────────────────────────────────
  48 ┤
  46 ┤
  44 ┤
  42 ┤
  40 ┤
  38 ┤
  36 ┤
  34 ┤
  32 ┤
  30 ┤
  28 ┤
  26 ┤
  24 ┤
  22 ┤
  20 ┤
  18 ┤
  16 ┤
  14 ┤
  12 ┤
  10 ┤
   8 ┤
   6 ┤
   4 ┤
   2 ┤
   0 ┤
Memory utilization (% of total, peak 25.00%)
 100 ┤
  98 ┤
  96 ┤
  94 ┤
  92 ┤
  90 ┤
  88 ┤
  86 ┤
  84 ┤
  82 ┤
  80 ┤
  78 ┤
  76 ┤
  74 ┤
  72 ┤
  70 ┤
  68 ┤
  66 ┤
  64 ┤
  62 ┤
  60 ┤
  58 ┤
  56 ┤
  54 ┤
  52 ┤
  50 ┤
  48 ┤
  46 ┤
  44 ┤
  42 ┤
  40 ┤
  38 ┤
  36 ┤
  34 ┤
  32 ┤
  30 ┤
  28 ┤
  26 ┼───────────────────────────────────────
  24 ┤
  22 ┤
  20 ┤
  18 ┤
  16 ┤
  14 ┤
  12 ┤
  10 ┤
   8 ┤
   6 ┤
   4 ┤
   2 ┤
   0 ┤
Legend: y-axis is utilization percent`

	require.Equal(t, expected, chart)
}

func TestRenderUtilizationChartFakeMultilineText(t *testing.T) {
	graph := renderUtilizationChart([]float64{5, 55, 20, 80, 30, 95, 25, 70, 15, 60, 10, 85, 5}, 13, 8, githubActionsChartMaxWidth)
	expected := ` 100 ┤    ╭╮
  88 ┤    ││    ╭╮
  75 ┤  ╭╮││╭╮  ││
  62 ┤  ││││││╭╮││
  50 ┤╭╮││││││││││
  38 ┤││││││││││││
  25 ┤│╰╯╰╯╰╯│││││
  12 ┤│      ╰╯╰╯│
   0 ┼╯          ╰`

	require.Equal(t, expected, graph)
}
