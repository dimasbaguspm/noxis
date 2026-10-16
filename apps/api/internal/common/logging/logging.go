// Package logging wires slog + OTLP; the ctx-aware handler stamps
// trace_id/span_id into every record.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"noxis/internal/common/config"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Log attribute keys.
const (
	serviceFallback = "noxis"
	// attribute keys stay consistent across log/trace/metric
	attrCollector = "collector"
	attrService   = "service"
)

// Shutdown flushes telemetry; call on process exit.
type Shutdown func(context.Context) error

func New(ctx context.Context, cfg config.Config) (*slog.Logger, Shutdown, error) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(&ctxHandler{inner: jsonHandler})
	slog.SetDefault(logger)

	if cfg.OTELCollector == "" {
		noop := func(context.Context) error { return nil }
		return logger, noop, nil
	}

	service := firstNonEmpty(cfg.OTELName, serviceFallback)
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

	logger.InfoContext(ctx, "telemetry connected", attrCollector, cfg.OTELCollector, attrService, service)
	return logger, tp.Shutdown, nil
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
