package executor

import (
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
		CpuChart: []*api.ChartPoint{
			{SecondsFromStart: 5, Value: 1.25},
			{SecondsFromStart: 10, Value: 2.0},
		},
		MemoryChart: []*api.ChartPoint{
			{SecondsFromStart: 3, Value: 1_000_000_000},
			{SecondsFromStart: 8, Value: 2_000_000_000},
		},
	}

	notice := formatGithubActionsNotice(snapshot, utilization)

	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 2.00 cores (50.00% of 4.00) at 10s%0APeak memory utilization: 2.0 GB (50.00% of 4.0 GB) at 8s", notice)
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

	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 1.50 cores (37.50% of 4.00) at 2s%0APeak memory utilization: 1.0 GB (25.00% of 4.0 GB) at 4s\n::warning title=Resource Utilization::Peak CPU and memory utilization are below 50% of available resources; it might be worth using a different resource class if possible.", notice)
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

	require.Equal(t, "::notice title=Resource Utilization::Peak CPU utilization: 1.50 cores (37.50% of 4.00) at 2s%0APeak memory utilization: 1.0 GB (25.00% of 4.0 GB) at 4s", notice)
}
