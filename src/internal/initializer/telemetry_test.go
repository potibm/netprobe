package initializer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type mockTraceExporter struct {
	shutdownCalled atomic.Bool
}

func (m *mockTraceExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	return nil
}

func (m *mockTraceExporter) Shutdown(_ context.Context) error {
	m.shutdownCalled.Store(true)

	return nil
}

type mockLogExporter struct {
	shutdownCalled atomic.Bool
}

func (m *mockLogExporter) Export(_ context.Context, _ []log.Record) error { return nil }

func (m *mockLogExporter) Shutdown(_ context.Context) error {
	m.shutdownCalled.Store(true)

	return nil
}

func (m *mockLogExporter) ForceFlush(_ context.Context) error { return nil }

type mockMetricExporter struct {
	shutdownCalled atomic.Bool
}

func (m *mockMetricExporter) Temporality(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (m *mockMetricExporter) Aggregation(_ sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.AggregationSum{}
}

func (m *mockMetricExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error {
	return nil
}

func (m *mockMetricExporter) ForceFlush(_ context.Context) error { return nil }

func (m *mockMetricExporter) Shutdown(_ context.Context) error {
	m.shutdownCalled.Store(true)

	return nil
}

func newMockTraceFactory() (*mockTraceExporter, traceFactoryFunc) {
	exp := &mockTraceExporter{}

	return exp, func(_ context.Context, _ string) (sdktrace.SpanExporter, error) {
		return exp, nil
	}
}

func newMockLogFactory() (*mockLogExporter, logFactoryFunc) {
	exp := &mockLogExporter{}

	return exp, func(_ context.Context, _ string) (log.Exporter, error) {
		return exp, nil
	}
}

func newMockMetricFactory() (*mockMetricExporter, metricFactoryFunc) {
	exp := &mockMetricExporter{}

	return exp, func(_ context.Context, _ string) (sdkmetric.Exporter, error) {
		return exp, nil
	}
}

func noopRuntimeStart(_ ...runtime.Option) error { return nil }

func TestInitTelemetry_EmptyEndpoint(t *testing.T) {
	shutdown, err := initTelemetry(context.Background(), "", "1.0.0", nil, nil, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, shutdown)
}

func TestInitTelemetry_Success(t *testing.T) {
	traceExp, traceFactory := newMockTraceFactory()
	logExp, logFactory := newMockLogFactory()
	metricExp, metricFactory := newMockMetricFactory()

	shutdown, err := initTelemetry(
		context.Background(),
		"localhost:4317",
		"1.0.0",
		traceFactory,
		logFactory,
		metricFactory,
		noopRuntimeStart,
	)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	shutdown()

	assert.True(t, traceExp.shutdownCalled.Load())
	assert.True(t, logExp.shutdownCalled.Load())
	assert.True(t, metricExp.shutdownCalled.Load())
}

func TestInitTelemetry_TraceExporterFails(t *testing.T) {
	failTrace := func(_ context.Context, _ string) (sdktrace.SpanExporter, error) {
		return nil, errors.New("trace exporter error")
	}

	shutdown, err := initTelemetry(
		context.Background(),
		"localhost:4317",
		"1.0.0",
		failTrace,
		nil,
		nil,
		noopRuntimeStart,
	)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "trace exporter error")
}

func TestInitTelemetry_LogExporterFails_CleansUpTrace(t *testing.T) {
	traceExp, traceFactory := newMockTraceFactory()

	failLog := func(_ context.Context, _ string) (log.Exporter, error) {
		return nil, errors.New("log exporter error")
	}

	shutdown, err := initTelemetry(
		context.Background(),
		"localhost:4317",
		"1.0.0",
		traceFactory,
		failLog,
		nil,
		noopRuntimeStart,
	)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
	assert.True(t, traceExp.shutdownCalled.Load(), "trace provider should be cleaned up")
}

func TestInitTelemetry_MetricExporterFails_CleansUpTraceAndLog(t *testing.T) {
	traceExp, traceFactory := newMockTraceFactory()
	logExp, logFactory := newMockLogFactory()

	failMetric := func(_ context.Context, _ string) (sdkmetric.Exporter, error) {
		return nil, errors.New("metric exporter error")
	}

	shutdown, err := initTelemetry(
		context.Background(),
		"localhost:4317",
		"1.0.0",
		traceFactory,
		logFactory,
		failMetric,
		noopRuntimeStart,
	)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
	assert.True(t, traceExp.shutdownCalled.Load(), "trace provider should be cleaned up")
	assert.True(t, logExp.shutdownCalled.Load(), "log provider should be cleaned up")
}

func TestInitTelemetry_RuntimeStartFails_CleansUpAll(t *testing.T) {
	traceExp, traceFactory := newMockTraceFactory()
	logExp, logFactory := newMockLogFactory()
	metricExp, metricFactory := newMockMetricFactory()

	failRuntime := func(_ ...runtime.Option) error {
		return errors.New("runtime start error")
	}

	shutdown, err := initTelemetry(
		context.Background(),
		"localhost:4317",
		"1.0.0",
		traceFactory,
		logFactory,
		metricFactory,
		failRuntime,
	)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
	assert.True(t, traceExp.shutdownCalled.Load(), "trace provider should be cleaned up")
	assert.True(t, logExp.shutdownCalled.Load(), "log provider should be cleaned up")
	assert.True(t, metricExp.shutdownCalled.Load(), "metric provider should be cleaned up")
}

func TestCleanupAll(t *testing.T) {
	var called [3]bool

	cleanups := []func(context.Context) error{
		func(_ context.Context) error {
			called[0] = true

			return nil
		},
		func(_ context.Context) error {
			called[1] = true

			return nil
		},
		func(_ context.Context) error {
			called[2] = true

			return nil
		},
	}

	cleanupAll(context.Background(), cleanups)

	assert.True(t, called[0])
	assert.True(t, called[1])
	assert.True(t, called[2])
}
