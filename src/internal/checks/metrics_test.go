package checks

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewProbeMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer mp.Shutdown(context.Background())

	// Set the global meter provider so otel.Meter() uses our test provider
	originalMeterProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer otel.SetMeterProvider(originalMeterProvider)

	logger := slog.New(slog.DiscardHandler)
	metrics := NewProbeMetrics(logger)

	assert.NotNil(t, metrics.ExecutionTotal, "ExecutionTotal should not be nil")
	assert.NotNil(t, metrics.Duration, "Duration should not be nil")
	assert.NotNil(t, metrics.Status, "Status should not be nil")

	// Verify the instruments are functional by recording some data
	ctx := context.Background()
	attrs := metric.WithAttributes(
		attribute.String("target_id", "test-target"),
		attribute.String("hostname", "example.com"),
		attribute.String("check_type", "HTTP"),
		attribute.String("interface", "eth0"),
	)

	metrics.ExecutionTotal.Add(ctx, 1, attrs)
	metrics.Duration.Record(ctx, 42, attrs)
	metrics.Status.Record(ctx, 1, attrs)

	// Collect the recorded metrics
	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Find our metrics in the collected data
	scopeMetrics := findScopeMetrics(t, rm, "github.com/potibm/netprobe/src/internal/checks")
	require.NotNil(t, scopeMetrics)

	assertMetricExists(t, scopeMetrics.Metrics, "netprobe.check.execution_total")
	assertMetricExists(t, scopeMetrics.Metrics, "netprobe.check.duration_milliseconds")
	assertMetricExists(t, scopeMetrics.Metrics, "netprobe.check.status")
}

func findScopeMetrics(t *testing.T, rm metricdata.ResourceMetrics, scopeName string) *metricdata.ScopeMetrics {
	t.Helper()
	for i := range rm.ScopeMetrics {
		if rm.ScopeMetrics[i].Scope.Name == scopeName {
			return &rm.ScopeMetrics[i]
		}
	}
	return nil
}

func assertMetricExists(t *testing.T, metrics []metricdata.Metrics, name string) {
	t.Helper()
	for _, m := range metrics {
		if m.Name == name {
			return
		}
	}
	t.Errorf("expected metric %s to exist in collected metrics", name)
}
