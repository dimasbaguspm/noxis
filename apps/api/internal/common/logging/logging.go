// Package logging wires slog + OTLP; the ctx-aware handler stamps
// trace_id/span_id into every record.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
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

// OTLP is the telemetry target logging needs. It is a local struct so this
// package never imports the config package — a sibling it must not depend on.
type OTLP struct {
	Collector string
	Name      string
}

func New(ctx context.Context, otlp OTLP) (*slog.Logger, Shutdown, error) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(&ctxHandler{inner: jsonHandler})
	slog.SetDefault(logger)
	// slog.SetDefault routes the standard log package (which OTel uses for its
	// internal reports) through this handler at info level; an exporter failure
	// is an error, so route them here instead.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.ErrorContext(context.Background(), "telemetry export failed", "error", err)
	}))
	// W3C trace context travels with requests: the middleware extracts a
	// caller's traceparent so one trace spans browser, API and database.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if otlp.Collector == "" {
		noop := func(context.Context) error { return nil }
		return logger, noop, nil
	}

	service := firstNonEmpty(otlp.Name, serviceFallback)
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(service),
	)

	traceExp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpointURL(otlp.Collector)))
	if err != nil {
		return nil, nil, fmt.Errorf("otel trace exporter: %w", err)
	}
	tp := sdkTrace.NewTracerProvider(sdkTrace.WithBatcher(traceExp), sdkTrace.WithResource(res))
	otel.SetTracerProvider(tp)

	logger.InfoContext(ctx, "telemetry connected", attrCollector, otlp.Collector, attrService, service)
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

// endpointURL makes NOXIS_OTEL_COLLECTOR a URL the OTLP/HTTP exporter accepts.
// It wants a scheme — https enables TLS, a bare host:port is plaintext http —
// and a path, because the exporter does not append the traces path to a URL,
// so a bare host is pointed at /v1/traces here. OTLP/HTTP rather than gRPC:
// a collector behind a proxy answers gRPC's export path with a 403.
func endpointURL(collector string) string {
	raw := collector
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/v1/traces"
	}
	return u.String()
}
