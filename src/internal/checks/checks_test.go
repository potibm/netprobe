package checks

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/potibm/netprobe/src/internal/config"
	"github.com/potibm/netprobe/src/internal/domain"
	netprobe_net "github.com/potibm/netprobe/src/internal/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type mockCheck struct {
	name       string
	result     domain.CheckResult
	execCalled atomic.Bool
}

func (m *mockCheck) Name() string { return m.name }

func (m *mockCheck) Execute(_ domain.ClientList, _ *slog.Logger) domain.CheckResult {
	m.execCalled.Store(true)

	return m.result
}

func setupTestMetrics(t *testing.T) (*sdkmetric.ManualReader, *sdkmetric.MeterProvider) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	orig := otel.GetMeterProvider()

	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(orig)

		_ = mp.Shutdown(context.Background())
	})

	return reader, mp
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics

	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)

	return rm
}

func findMetric(t *testing.T, rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}

	return nil
}

func TestHandleResult_Success(t *testing.T) {
	reader, _ := setupTestMetrics(t)

	metrics, err := NewProbeMetrics()
	require.NoError(t, err)

	cr := &CheckRunner{
		metrics: metrics,
		iface:   "eth0",
		logger:  slog.New(slog.DiscardHandler),
	}

	target := domain.Target{ID: "t1", Hostname: "example.com"}
	result := domain.CheckResult{
		CheckName: "HTTP",
		Success:   true,
		Duration:  100 * time.Millisecond,
	}

	cr.handleResult(context.Background(), target, result, cr.logger)

	rm := collectMetrics(t, reader)

	durationMetric := findMetric(t, rm, "netprobe.check.duration_milliseconds")
	require.NotNil(t, durationMetric, "duration metric should be recorded")

	execMetric := findMetric(t, rm, "netprobe.check.execution_total")
	require.NotNil(t, execMetric, "execution_total metric should be recorded")

	statusMetric := findMetric(t, rm, "netprobe.check.status")
	require.NotNil(t, statusMetric, "status metric should be recorded")
}

func TestHandleResult_Failure(t *testing.T) {
	reader, _ := setupTestMetrics(t)

	metrics, err := NewProbeMetrics()
	require.NoError(t, err)

	cr := &CheckRunner{
		metrics: metrics,
		iface:   "eth0",
		logger:  slog.New(slog.DiscardHandler),
	}

	target := domain.Target{ID: "t1", Hostname: "example.com"}
	result := domain.CheckResult{
		CheckName:    "HTTP",
		Success:      false,
		ErrorCode:    domain.ErrNotAvailable,
		ErrorMessage: "Service is offline",
		Duration:     50 * time.Millisecond,
	}

	cr.handleResult(context.Background(), target, result, cr.logger)

	rm := collectMetrics(t, reader)

	execMetric := findMetric(t, rm, "netprobe.check.execution_total")
	require.NotNil(t, execMetric, "execution_total metric should be recorded")

	statusMetric := findMetric(t, rm, "netprobe.check.status")
	require.NotNil(t, statusMetric, "status metric should be recorded")
}

func TestRunOnce(t *testing.T) {
	reader, _ := setupTestMetrics(t)

	metrics, err := NewProbeMetrics()
	require.NoError(t, err)

	check := &mockCheck{
		name: "test-check",
		result: domain.CheckResult{
			CheckName: "test-check",
			Success:   true,
		},
	}

	cr := &CheckRunner{
		targets: []domain.Target{
			{ID: "t1", Hostname: "example.com", Checks: []domain.Check{check}},
		},
		metrics: metrics,
		iface:   "eth0",
		logger:  slog.New(slog.DiscardHandler),
	}

	cr.runOnce(context.Background())

	assert.True(t, check.execCalled.Load(), "check should have been executed")

	rm := collectMetrics(t, reader)
	assert.NotNil(t, findMetric(t, rm, "netprobe.check.duration_milliseconds"))
}

func TestRun_ContextCancellation(t *testing.T) {
	metrics, err := NewProbeMetrics()
	require.NoError(t, err)

	cr := &CheckRunner{
		targets: []domain.Target{},
		cfg:     config.Defaults{IntervalSeconds: 1},
		metrics: metrics,
		iface:   "eth0",
		logger:  slog.New(slog.DiscardHandler),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		cr.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestNewCheckRunner_IP4Only(t *testing.T) {
	cfg := config.Config{
		Name: "test",
		Defaults: config.Defaults{
			TimeoutSeconds:  5,
			IntervalSeconds: 10,
		},
	}

	logger := slog.New(slog.DiscardHandler)

	runner, err := NewCheckRunner(cfg, "nonexistent0", netprobe_net.IP4, logger)
	require.NoError(t, err)
	require.NotNil(t, runner)

	assert.Len(t, runner.clients, 0, "invalid interface should not create clients")
}

func TestNewCheckRunner_InvalidInterface(t *testing.T) {
	cfg := config.Config{
		Name: "test",
		Defaults: config.Defaults{
			TimeoutSeconds:  5,
			IntervalSeconds: 10,
		},
	}

	logger := slog.New(slog.DiscardHandler)

	runner, err := NewCheckRunner(cfg, "nonexistent0", netprobe_net.IPAuto, logger)
	require.NoError(t, err)
	require.NotNil(t, runner)

	assert.Len(t, runner.clients, 0, "invalid interface should not create clients")
}

func TestHandleResult_MetricsAttributes(t *testing.T) {
	reader, _ := setupTestMetrics(t)

	metrics, err := NewProbeMetrics()
	require.NoError(t, err)

	cr := &CheckRunner{
		metrics: metrics,
		iface:   "eth0",
		logger:  slog.New(slog.DiscardHandler),
	}

	target := domain.Target{ID: "target-1", Hostname: "test.local"}
	result := domain.CheckResult{
		CheckName: "HTTP",
		Success:   true,
		Duration:  42 * time.Millisecond,
	}

	cr.handleResult(context.Background(), target, result, cr.logger)

	rm := collectMetrics(t, reader)

	execMetric := findMetric(t, rm, "netprobe.check.execution_total")
	require.NotNil(t, execMetric)

	sum, ok := execMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.Len(t, sum.DataPoints, 1)

	dp := sum.DataPoints[0]
	assert.Equal(t, int64(1), dp.Value)

	attrs := dp.Attributes.ToSlice()
	assert.Contains(t, attrs, attribute.String("target_id", "target-1"))
	assert.Contains(t, attrs, attribute.String("hostname", "test.local"))
	assert.Contains(t, attrs, attribute.String("check_type", "HTTP"))
	assert.Contains(t, attrs, attribute.String("interface", "eth0"))
	assert.Contains(t, attrs, attribute.String("status", "success"))
}
