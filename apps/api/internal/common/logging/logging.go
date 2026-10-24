// Package logging wires slog and OTLP, adding trace ids to stdout records when no collector is configured.
package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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

// Signal paths the OTLP/HTTP exporters post to.
const (
	pathTraces  = "/v1/traces"
	pathLogs    = "/v1/logs"
	pathMetrics = "/v1/metrics"
)

// Shutdown flushes telemetry; call on process exit.
type Shutdown func(context.Context) error

// OTLP is the telemetry target logging needs. It is a local struct so this package never imports the config package — a sibling it must not depend on.
type OTLP struct {
	Collector string
	Name      string
}

func New(ctx context.Context, otlp OTLP) (*slog.Logger, Shutdown, error) {
	// A telemetry failure cannot be reported through the collector that just failed, so it goes to stderr where an operator always sees it.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).
			ErrorContext(context.Background(), "telemetry export failed", "error", err)
	}))
	// W3C trace context travels with requests: the middleware extracts a caller's traceparent so one trace spans browser, API and database.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// No collector: nothing to export to, so records stay on stdout.
	if otlp.Collector == "" {
		logger := stdoutLogger()
		slog.SetDefault(logger)
		return logger, noShutdown, nil
	}

	service := firstNonEmpty(otlp.Name, serviceFallback)
	res := newResource(service)

	traceExp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(signalURL(otlp.Collector, pathTraces)))
	if err != nil {
		return nil, nil, fmt.Errorf("otel trace exporter: %w", err)
	}
	tp := sdkTrace.NewTracerProvider(sdkTrace.WithBatcher(traceExp), sdkTrace.WithResource(res))
	otel.SetTracerProvider(tp)

	// Records carry their trace and span ids without help: the log SDK reads the active span off the record's context.
	logExp, err := otlploghttp.New(ctx, otlploghttp.WithEndpointURL(signalURL(otlp.Collector, pathLogs)))
	if err != nil {
		return nil, nil, fmt.Errorf("otel log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)

	// Metrics share the collector and its cadence.
	metricExp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(signalURL(otlp.Collector, pathMetrics)))
	if err != nil {
		return nil, nil, fmt.Errorf("otel metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// With a collector configured, records go there and not to stdout.
	logger := slog.New(&otelHandler{logger: lp.Logger(service)})
	slog.SetDefault(logger)

	logger.InfoContext(ctx, "telemetry connected", attrCollector, otlp.Collector, attrService, service)
	return logger, func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), lp.Shutdown(ctx), mp.Shutdown(ctx))
	}, nil
}

// stdoutLogger is the fallback when no collector is configured: JSON on stdout with the active span's ids stamped in.
func stdoutLogger() *slog.Logger {
	return slog.New(&ctxHandler{inner: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})})
}

// newResource describes this process to every signal: the service, its build version and the host it runs on.
func newResource(service string) *resource.Resource {
	attrs := []attribute.KeyValue{semconv.ServiceName(service)}
	if v := buildVersion(); v != "" {
		attrs = append(attrs, semconv.ServiceVersion(v))
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		attrs = append(attrs, semconv.HostName(h))
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attrs...)
}

// buildVersion is the main module's version when the binary carries module information, empty otherwise.
func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return ""
}

// ctxHandler injects the active span's ids into every record whose context carries one. Only the stdout fallback needs it.
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

// otelHandler forwards records to the OTel log pipeline.
type otelHandler struct {
	logger otellog.Logger
	attrs  []attribute.KeyValue
	groups []string
}

func (h *otelHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	var rec otellog.Record
	rec.SetTimestamp(r.Time)
	rec.SetObservedTimestamp(time.Now())
	rec.SetSeverity(severity(r.Level))
	rec.SetSeverityText(r.Level.String())
	rec.SetBody(attribute.StringValue(r.Message))

	kvs := make([]attribute.KeyValue, 0, len(h.attrs)+r.NumAttrs())
	kvs = append(kvs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		kvs = append(kvs, flatten(h.groups, a)...)
		return true
	})
	rec.AddAttributes(kvs...)

	h.logger.Emit(ctx, rec)
	return nil
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{
		logger: h.logger,
		attrs:  append(append([]attribute.KeyValue{}, h.attrs...), flatten(h.groups, attrs...)...),
		groups: h.groups,
	}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &otelHandler{
		logger: h.logger,
		attrs:  h.attrs,
		groups: append(append([]string{}, h.groups...), name),
	}
}

// flatten turns slog attributes into OTel key-values, prefixing the group path and recursing into nested groups.
func flatten(groups []string, attrs ...slog.Attr) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		if a.Equal(slog.Attr{}) {
			continue
		}
		if a.Value.Kind() == slog.KindGroup {
			out = append(out, flatten(append(append([]string{}, groups...), a.Key), a.Value.Group()...)...)
			continue
		}
		key := a.Key
		if len(groups) > 0 {
			key = strings.Join(append(append([]string{}, groups...), a.Key), ".")
		}
		out = append(out, attribute.KeyValue{Key: attribute.Key(key), Value: value(a.Value)})
	}
	return out
}

// value keeps a record's attribute types instead of flattening everything to a string.
func value(v slog.Value) attribute.Value {
	switch v.Kind() {
	case slog.KindBool:
		return attribute.BoolValue(v.Bool())
	case slog.KindInt64:
		return attribute.Int64Value(v.Int64())
	case slog.KindUint64:
		return attribute.Int64Value(int64(v.Uint64()))
	case slog.KindFloat64:
		return attribute.Float64Value(v.Float64())
	case slog.KindDuration:
		return attribute.Int64Value(v.Duration().Nanoseconds())
	case slog.KindTime:
		return attribute.Int64Value(v.Time().UnixNano())
	default:
		return attribute.StringValue(v.String())
	}
}

func severity(l slog.Level) otellog.Severity {
	switch {
	case l < slog.LevelInfo:
		return otellog.SeverityDebug
	case l < slog.LevelWarn:
		return otellog.SeverityInfo
	case l < slog.LevelError:
		return otellog.SeverityWarn
	default:
		return otellog.SeverityError
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func noShutdown(context.Context) error { return nil }

// signalURL adds an HTTP scheme and signal path to a bare OTLP collector host.
func signalURL(collector, path string) string {
	raw := collector
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = path
	}
	return u.String()
}
