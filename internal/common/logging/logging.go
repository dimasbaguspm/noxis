// Package logging builds the process log sink and, when an OTEL collector is
// configured, real OTLP trace + metric pipelines with a service resource.
//
// Every handler is context-aware: log records emitted from a context carrying
// an active span get trace_id/span_id attributes, so logs, traces and metrics
// share one identity. Call sites must use the *Context variants
// (slog.InfoContext, app.Log.InfoContext, ...) — the ctx is the carrier.
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
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"noxis/internal/entity"
)

// Shutdown flushes telemetry; call on process exit.
type Shutdown func(context.Context) error

func New(cfg entity.Config) (*slog.Logger, Shutdown, error) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(&ctxHandler{inner: jsonHandler})
	slog.SetDefault(logger)

	if cfg.OTELCollector == "" {
		return logger, func(context.Context) error { return nil }, nil
	}

	ctx := context.Background()
	service := firstNonEmpty(cfg.OTELName, "noxis")
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(service),
	)

	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.OTELCollector), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("otel trace exporter: %w", err)
	}
	tp := sdkTrace.NewTracerProvider(sdkTrace.WithBatcher(traceExp), sdkTrace.WithResource(res))
	otel.SetTracerProvider(tp)

	metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(cfg.OTELCollector), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("otel metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metricExp)), metric.WithResource(res))
	otel.SetMeterProvider(mp)

	logger.InfoContext(ctx, "telemetry connected", "collector", cfg.OTELCollector, "service", service)
	return logger, func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		return mp.Shutdown(ctx)
	}, nil
}

// ctxHandler injects the active span's ids into every record whose context
// carries one.
type ctxHandler struct {
	inner slog.Handler
}

func (h *ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ctxHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ctxHandler) WithGroup(name string) slog.Handler {
	return &ctxHandler{inner: h.inner.WithGroup(name)}
}

func (h *ctxHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
