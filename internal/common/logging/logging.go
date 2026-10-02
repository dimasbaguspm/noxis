// Package logging builds the process log sink and, when an OTEL collector is
// configured, real OTLP trace + metric pipelines with a service resource.
// Logs stay slog JSON with OTEL-compatible attributes; spans/metrics flow to
// the collector. gRPC servers/clients are instrumented in cmd bootstrap.
//
// ponytail: trace_id/span_id injected into log records (OTEL Logs SDK) is a
// known ceiling — spans already link to logs via service name + timestamps.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"noxis/internal/common/config"
)

// Shutdown flushes telemetry; call on process exit.
type Shutdown func(context.Context) error

func New(cfg config.Config) (*slog.Logger, Shutdown, error) {
	attrs := []slog.Attr{
		slog.String("otel.name", cfg.OTELName),
		slog.String("service.name", firstNonEmpty(cfg.OTELName, "noxis")),
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}).WithAttrs(attrs))
	slog.SetDefault(logger)

	if cfg.OTELCollector == "" {
		return logger, func(context.Context) error { return nil }, nil
	}

	ctx := context.Background()
	res := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(firstNonEmpty(cfg.OTELName, "noxis")))

	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.OTELCollector), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("otel trace exporter: %w", err)
	}
	tp := trace.NewTracerProvider(trace.WithBatcher(traceExp), trace.WithResource(res))
	otel.SetTracerProvider(tp)

	metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(cfg.OTELCollector), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("otel metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metricExp)), metric.WithResource(res))
	otel.SetMeterProvider(mp)

	logger.Info("telemetry connected", "collector", cfg.OTELCollector, "service", cfg.OTELName)
	return logger, func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		return mp.Shutdown(ctx)
	}, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}