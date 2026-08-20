package initializer

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type (
	traceFactoryFunc  func(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error)
	logFactoryFunc    func(ctx context.Context, endpoint string) (sdklog.Exporter, error)
	metricFactoryFunc func(ctx context.Context, endpoint string) (sdkmetric.Exporter, error)
	runtimeStartFunc  func(opts ...runtime.Option) error
)

func InitTelemetry(ctx context.Context, endpoint, version string) (func(), error) {
	return initTelemetry(
		ctx,
		endpoint,
		version,
		func(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
			return otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint(endpoint))
		},
		func(ctx context.Context, endpoint string) (sdklog.Exporter, error) {
			return otlploggrpc.New(ctx, otlploggrpc.WithInsecure(), otlploggrpc.WithEndpoint(endpoint))
		},
		func(ctx context.Context, endpoint string) (sdkmetric.Exporter, error) {
			return otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure(), otlpmetricgrpc.WithEndpoint(endpoint))
		},
		runtime.Start,
	)
}

func initTelemetry(
	ctx context.Context,
	endpoint, version string,
	newTraceExporter traceFactoryFunc,
	newLogExporter logFactoryFunc,
	newMetricExporter metricFactoryFunc,
	startRuntime runtimeStartFunc,
) (func(), error) {
	if endpoint == "" {
		slog.Warn("No OpenTelemetry endpoint configured, telemetry will be disabled")

		return nil, nil
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName("netprobe"),
		semconv.ServiceVersion(version),
	))
	if err != nil {
		return nil, err
	}

	var cleanups []func(context.Context) error

	traceExporter, err := newTraceExporter(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	cleanups = append(cleanups, tp.Shutdown)

	logExporter, err := newLogExporter(ctx, endpoint)
	if err != nil {
		cleanupAll(ctx, cleanups)

		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)
	cleanups = append(cleanups, lp.Shutdown)

	metricExporter, err := newMetricExporter(ctx, endpoint)
	if err != nil {
		cleanupAll(ctx, cleanups)

		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	cleanups = append(cleanups, mp.Shutdown)

	err = startRuntime(runtime.WithMeterProvider(mp))
	if err != nil {
		cleanupAll(ctx, cleanups)

		return nil, err
	}

	return func() {
		cleanupAll(ctx, cleanups)
	}, nil
}

func cleanupAll(ctx context.Context, cleanups []func(context.Context) error) {
	const shutdownTimeout = 5 * time.Second

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	for _, fn := range cleanups {
		_ = fn(shutdownCtx)
	}
}
