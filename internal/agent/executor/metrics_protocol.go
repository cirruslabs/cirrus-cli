package executor

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/omni-cache/pkg/protocols"
	"github.com/dustin/go-humanize"
)

type metricsProtocolFactory struct {
	collector *metrics.Collector
}

func (metricsProtocolFactory) ID() string {
	return "cirrus-metrics"
}

func (factory metricsProtocolFactory) New(_ protocols.Dependencies) (protocols.Protocol, error) {
	return &metricsProtocol{collector: factory.collector}, nil
}

type metricsProtocol struct {
	collector *metrics.Collector
}

func (protocol *metricsProtocol) Register(registrar *protocols.Registrar) error {
	registrar.HTTP().HandleFunc("GET /metrics", protocol.handleMetrics)
	return nil
}

func (protocol *metricsProtocol) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if protocol.collector == nil {
		http.Error(w, "metrics collector unavailable", http.StatusServiceUnavailable)
		return
	}

	utilization := protocol.collector.ResourceUtilizationSnapshot()
	if acceptsJSON(r.Header.Get("Accept")) {
		w.Header().Set("Content-Type", "application/json")
		response := metricsResponse{
			Snapshot:            snapshotToResponse(protocol.collector.Snapshot()),
			ResourceUtilization: utilization,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.ErrorContext(r.Context(), "failed to encode metrics response", "err", err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, formatMetricsSummary(protocol.collector.Snapshot(), utilization))
}

type metricsResponse struct {
	Snapshot            metricsSnapshot          `json:"snapshot"`
	ResourceUtilization *api.ResourceUtilization `json:"resource_utilization,omitempty"`
}

type metricsSnapshot struct {
	Timestamp   string  `json:"timestamp,omitempty"`
	CPUUsed     float64 `json:"cpu_used"`
	MemoryUsed  float64 `json:"memory_used"`
	CPUTotal    float64 `json:"cpu_total"`
	MemoryTotal float64 `json:"memory_total"`
	CPUError    string  `json:"cpu_error,omitempty"`
	MemoryError string  `json:"memory_error,omitempty"`
	TotalsError string  `json:"totals_error,omitempty"`
}

func snapshotToResponse(snapshot metrics.Snapshot) metricsSnapshot {
	response := metricsSnapshot{
		CPUUsed:     snapshot.CPUUsed,
		MemoryUsed:  snapshot.MemoryUsed,
		CPUTotal:    snapshot.CPUTotal,
		MemoryTotal: snapshot.MemoryTotal,
	}

	if !snapshot.Timestamp.IsZero() {
		response.Timestamp = snapshot.Timestamp.Format("2006-01-02T15:04:05Z07:00")
	}
	if snapshot.CPUError != nil {
		response.CPUError = snapshot.CPUError.Error()
	}
	if snapshot.MemoryError != nil {
		response.MemoryError = snapshot.MemoryError.Error()
	}
	if snapshot.TotalsError != nil {
		response.TotalsError = snapshot.TotalsError.Error()
	}

	return response
}

func formatMetricsSummary(snapshot metrics.Snapshot, utilization *api.ResourceUtilization) string {
	cpuTotal := snapshot.CPUTotal
	memoryTotal := snapshot.MemoryTotal
	if utilization != nil {
		if cpuTotal == 0 {
			cpuTotal = utilization.CpuTotal
		}
		if memoryTotal == 0 {
			memoryTotal = utilization.MemoryTotal
		}
	}

	cpuPercent := snapshot.CPUUsed * 100.0
	if cpuTotal > 0 {
		cpuPercent = (snapshot.CPUUsed / cpuTotal) * 100.0
	}

	var builder strings.Builder
	builder.WriteString("agent metrics\n")
	fmt.Fprintf(&builder, "cpu: %.2f cores (%.2f%%", snapshot.CPUUsed, cpuPercent)
	if cpuTotal > 0 {
		fmt.Fprintf(&builder, " of %.2f", cpuTotal)
	}
	builder.WriteString(")\n")

	memoryUsed := uint64(maxFloat(snapshot.MemoryUsed, 0))
	if memoryTotal > 0 {
		memoryPercent := (snapshot.MemoryUsed / memoryTotal) * 100.0
		fmt.Fprintf(&builder, "memory: %s / %s (%.2f%%)\n",
			humanize.Bytes(memoryUsed),
			humanize.Bytes(uint64(memoryTotal)),
			memoryPercent)
	} else {
		fmt.Fprintf(&builder, "memory: %s\n", humanize.Bytes(memoryUsed))
	}

	if utilization != nil {
		fmt.Fprintf(&builder, "points: cpu=%d memory=%d\n", len(utilization.CpuChart), len(utilization.MemoryChart))
	}

	if snapshot.CPUError != nil || snapshot.MemoryError != nil || snapshot.TotalsError != nil {
		fmt.Fprintf(&builder, "errors: cpu=%t memory=%t totals=%t\n",
			snapshot.CPUError != nil,
			snapshot.MemoryError != nil,
			snapshot.TotalsError != nil)
	}

	return builder.String()
}

func acceptsJSON(acceptHeader string) bool {
	if strings.TrimSpace(acceptHeader) == "" {
		return false
	}
	for _, part := range strings.Split(acceptHeader, ",") {
		mediaType := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
			return true
		}
	}
	return false
}

func maxFloat(value float64, min float64) float64 {
	if value < min {
		return min
	}
	return value
}
