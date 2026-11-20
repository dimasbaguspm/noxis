// Package telemetry builds the one set of providers this process uses. With an OTLP endpoint
// configured, traces, metrics and logs leave the process; without one, everything is a no-op and the
// binary behaves exactly as it does with no collector at all. It is the only place that knows OTLP.
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"noxis/internal/common/config"
)

// instruments are the counters and histograms this build records. They are created once, by New, and
// are nil until then, so a caller never has to know whether a collector is configured.
type instruments struct {
	rehearsalSeconds metric.Float64Histogram
	rehearsalVerdict metric.Int64Counter
	applySeconds     metric.Float64Histogram
	applyOutcome     metric.Int64Counter
}

// FlushTimeout bounds the last chance to push a batched span out of the process.
const FlushTimeout = 5 * time.Second

// measures holds the instruments once New has built them.
var measures *instruments

// Closer is what New hands back: closing it flushes what is still in flight.
type Closer func(context.Context) error

// New builds a logger and the providers. An empty endpoint returns a plain slog logger and a no-op
// closer, which is the development and test path.
//
// The level and the format are constants: a log line is for a machine to ship and a human to grep,
// and neither of those is a decision a deployment makes per environment. JSON at Info, always.
func New(ctx context.Context, cfg config.Config) (*slog.Logger, Closer, error) {
	stdio := func() slog.Handler {
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	if cfg.OTELEndpoint == "" {
		return slog.New(stdio()), func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.OTELServiceName)),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry: resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry: trace exporter: %w", err)
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry: metric exporter: %w", err)
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	)

	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry: log exporter: %w", err)
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(loggerProvider)

	if err := measure(meterProvider); err != nil {
		return nil, nil, err
	}

	logger := slog.New(otelslog.NewHandler(cfg.OTELServiceName,
		otelslog.WithLoggerProvider(loggerProvider), otelslog.WithSource(true)))

	shutdown := func(ctx context.Context) error {
		var firstErr error
		for _, fn := range []func(context.Context) error{
			meterProvider.Shutdown, loggerProvider.Shutdown, tracerProvider.Shutdown,
		} {
			if err := fn(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}
	return logger, shutdown, nil
}

// measure creates the instruments once: a counter that is created per request is a counter nobody can
// aggregate.
func measure(provider *sdkmetric.MeterProvider) error {
	meter := provider.Meter("noxis")
	var err error
	built := &instruments{}
	if built.rehearsalSeconds, err = meter.Float64Histogram("noxis.rehearsal.duration",
		metric.WithUnit("s"), metric.WithDescription("how long a rehearsal took")); err != nil {
		return fmt.Errorf("telemetry: rehearsal histogram: %w", err)
	}
	if built.rehearsalVerdict, err = meter.Int64Counter("noxis.rehearsal.verdict",
		metric.WithDescription("rehearsals by tier and verdict")); err != nil {
		return fmt.Errorf("telemetry: rehearsal counter: %w", err)
	}
	if built.applySeconds, err = meter.Float64Histogram("noxis.apply.duration",
		metric.WithUnit("s"), metric.WithDescription("how long an apply took")); err != nil {
		return fmt.Errorf("telemetry: apply histogram: %w", err)
	}
	if built.applyOutcome, err = meter.Int64Counter("noxis.apply.outcome",
		metric.WithDescription("applies by outcome")); err != nil {
		return fmt.Errorf("telemetry: apply counter: %w", err)
	}
	measures = built
	return nil
}

// Span starts one span named after the thing it measures — `Package.Type.Method`, or the route it
// serves. The returned function ends it and records the error, if any.
func Span(ctx context.Context, name string, attributes ...attribute.KeyValue) (context.Context, func(error)) {
	ctx, span := otel.Tracer("noxis").Start(ctx, name, trace.WithAttributes(attributes...))
	return ctx, func(err error) {
		RecordError(span, err)
		span.End()
	}
}

// Rehearsal records one rehearsal's duration and verdict. The labels are deliberately coarse: a tier,
// a verdict and a driver, never a ref.
func Rehearsal(ctx context.Context, tier, verdict, driver string, seconds float64) {
	if measures == nil {
		return
	}
	set := metric.WithAttributes(attribute.String("tier", tier), attribute.String("driver", driver))
	measures.rehearsalSeconds.Record(ctx, seconds, set)
	measures.rehearsalVerdict.Add(ctx, 1, metric.WithAttributes(
		attribute.String("tier", tier), attribute.String("verdict", verdict), attribute.String("driver", driver)))
}

// Apply records one apply's duration and outcome.
func Apply(ctx context.Context, outcome, driver string, seconds float64) {
	if measures == nil {
		return
	}
	set := metric.WithAttributes(attribute.String("driver", driver))
	measures.applySeconds.Record(ctx, seconds, set)
	measures.applyOutcome.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome), attribute.String("driver", driver)))
}

// RecordError stamps an error on a span without ending it.
func RecordError(span trace.Span, err error) {
	if err == nil || !span.IsRecording() {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// Outcome names one apply's ending in two words, so a dashboard can group by it.
func Outcome(err error) string {
	if err != nil {
		return "failed"
	}
	return "ok"
}
