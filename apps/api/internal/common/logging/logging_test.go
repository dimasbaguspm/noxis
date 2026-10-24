package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/codes"

	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	otellog "go.opentelemetry.io/otel/log"

	"go.opentelemetry.io/otel"
)

// TestSignalURL pins the forms NOXIS_OTEL_COLLECTOR accepts: a full URL (https enables TLS) and a bare host:port, which stays plaintext.
func TestSignalURL(t *testing.T) {
	cases := []struct {
		collector string
		path      string
		want      string
	}{
		{"collector:4318", pathLogs, "http://collector:4318/v1/logs"},
		{"http://collector:4318", pathTraces, "http://collector:4318/v1/traces"},
		{"https://otel.dimasbaguspm.dev", pathTraces, "https://otel.dimasbaguspm.dev/v1/traces"},
		{"https://otel.dimasbaguspm.dev", pathLogs, "https://otel.dimasbaguspm.dev/v1/logs"},
		{"https://otel.example.com/", pathTraces, "https://otel.example.com/v1/traces"},
		{"https://otel.example.com/custom", pathTraces, "https://otel.example.com/custom"},
	}
	for _, c := range cases {
		if got := signalURL(c.collector, c.path); got != c.want {
			t.Errorf("signalURL(%q, %q) = %q, want %q", c.collector, c.path, got, c.want)
		}
	}
}

// TestSeverity maps every slog level onto the OTel scale.
func TestSeverity(t *testing.T) {
	cases := map[slog.Level]otellog.Severity{
		slog.LevelDebug: otellog.SeverityDebug,
		slog.LevelInfo:  otellog.SeverityInfo,
		slog.LevelWarn:  otellog.SeverityWarn,
		slog.LevelError: otellog.SeverityError,
	}
	for in, want := range cases {
		if got := severity(in); got != want {
			t.Errorf("severity(%v) = %v, want %v", in, got, want)
		}
	}
}

// TestFlatten prefixes the group path and keeps the attribute types.
func TestFlatten(t *testing.T) {
	kvs := flatten([]string{"http"}, slog.String("method", "GET"), slog.Int("status", 500))
	if len(kvs) != 2 {
		t.Fatalf("got %d attrs, want 2", len(kvs))
	}
	if kvs[0].Key != "http.method" || kvs[0].Value.AsString() != "GET" {
		t.Errorf("first = %+v", kvs[0])
	}
	if kvs[1].Key != "http.status" || kvs[1].Value.AsInt64() != 500 {
		t.Errorf("second = %+v", kvs[1])
	}
}

// TestFlattenNestedGroup recurses into a group attribute.
func TestFlattenNestedGroup(t *testing.T) {
	kvs := flatten(nil, slog.Group("db", slog.String("system", "postgres")))
	if len(kvs) != 1 || kvs[0].Key != "db.system" {
		t.Fatalf("got %+v", kvs)
	}
}

// A decision is recorded twice: as a span event for the trace and as a log record at the requested level, so either sink alone explains it.
func TestInfoAndWarnRecordSpanEventAndLog(t *testing.T) {
	for _, tc := range []struct {
		level slog.Level
		emit  func(context.Context, string, ...any)
		want  string
	}{
		{slog.LevelInfo, InfoWithCtx, `"level":"INFO"`},
		{slog.LevelWarn, WarnWithCtx, `"level":"WARN"`},
	} {
		recorder := newRecorder(t)
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		t.Cleanup(func() { slog.SetDefault(prev) })

		ctx, end := StartSpan(context.Background(), "Test.Decision")
		tc.emit(ctx, "authz denied", "connection", "demo", "need", "write")
		end(nil)

		spans := recorder.Ended()
		if len(spans) != 1 || len(spans[0].Events()) != 1 || spans[0].Events()[0].Name != "authz denied" {
			t.Fatalf("span events = %+v, want one authz denied", spans)
		}
		line := buf.String()
		if !strings.Contains(line, "authz denied") || !strings.Contains(line, tc.want) || !strings.Contains(line, `"connection":"demo"`) {
			t.Fatalf("log %s missing the decision", line)
		}
	}
}

// A failure marks the span failed and keeps the wrapped cause in the log.
func TestErrorWithCtxMarksSpanAndKeepsCause(t *testing.T) {
	recorder := newRecorder(t)
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	ctx, end := StartSpan(context.Background(), "Test.Failure")
	err := fmt.Errorf("apply change: %w", errors.New("duplicate key"))
	ErrorWithCtx(ctx, "change failed to apply", err, "change", "x")
	end(nil)

	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Status().Code != codes.Error {
		t.Fatalf("span status = %+v, want error", spans)
	}
	if line := buf.String(); !strings.Contains(line, "duplicate key") || !strings.Contains(line, `"level":"ERROR"`) {
		t.Fatalf("log %s missing the cause", line)
	}
}

// newRecorder installs a recording tracer provider for one test.
func newRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	rec := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdkTrace.NewTracerProvider(sdkTrace.WithSpanProcessor(rec)))
	return rec
}
