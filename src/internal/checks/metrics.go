package checks

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type ProbeMetrics struct {
	ExecutionTotal metric.Int64Counter
	Duration       metric.Int64Histogram
	Status         metric.Int64Gauge
}

func NewProbeMetrics() (ProbeMetrics, error) {
	meter := otel.Meter("github.com/potibm/netprobe/src/internal/checks")

	execTotal, err := meter.Int64Counter(
		"netprobe.check.execution_total",
		metric.WithDescription("Total number of executed checks"),
	)
	if err != nil {
		return ProbeMetrics{}, err
	}

	duration, err := meter.Int64Histogram(
		"netprobe.check.duration_milliseconds",
		metric.WithDescription("Latency of the check in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return ProbeMetrics{}, err
	}

	status, err := meter.Int64Gauge(
		"netprobe.check.status",
		metric.WithDescription("Current status of the target (1=Ok, 0=Error/Alert)"),
	)
	if err != nil {
		return ProbeMetrics{}, err
	}

	return ProbeMetrics{
		ExecutionTotal: execTotal,
		Duration:       duration,
		Status:         status,
	}, nil
}
