package middlewares

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// RequestLog records the request line including the status code written by
// the handler, even when no explicit WriteHeader call sets it.
func TestRequestLogRecordsStatus(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/external/v1/engine/sql/connections", http.NoBody)
	RequestLog(inner, log).ServeHTTP(rec, req)

	line := buf.String()
	for _, want := range []string{`"method":"GET"`, `"path":"/external/v1/engine/sql/connections"`, `"status":404`} {
		if !strings.Contains(line, want) {
			t.Fatalf("log %s missing %s", line, want)
		}
	}
}

// The request log carries the authenticated caller's attributes once, from
// the context (the middleware that set the principal ran before this).
func TestRequestLogAttachesPrincipal(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{Subject: "bob", Groups: []string{"readers"}}))
	RequestLog(okHandler(), log).ServeHTTP(rec, req)

	line := buf.String()
	for _, want := range []string{`"user_id":"bob"`, `"groups":["readers"]`} {
		if !strings.Contains(line, want) {
			t.Fatalf("log %s missing %s", line, want)
		}
	}
}

// The span-attribute attach is a no-op without a recording tracer (the
// default SDK), so RequestLog must serve fine with no OTEL configured.
func TestRequestLogWithoutTracingDoesNotPanic(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Identity{Subject: "bob"}))
	RequestLog(okHandler(), log).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

// setupTracing installs a recording provider and the W3C propagator for the
// test, restoring the process globals afterwards.
func setupTracing(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdkTrace.NewTracerProvider(sdkTrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return recorder
}

// A caller's traceparent becomes the parent of the request span.
func TestTraceJoinsIncomingTrace(t *testing.T) {
	recorder := setupTracing(t)
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"

	var got trace.SpanContext
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = trace.SpanContextFromContext(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")

	Trace(inner).ServeHTTP(httptest.NewRecorder(), req)

	if got.TraceID().String() != traceID {
		t.Fatalf("handler span trace id=%s want=%s", got.TraceID(), traceID)
	}
	if spans := recorder.Ended(); len(spans) != 1 || spans[0].SpanContext().TraceID().String() != traceID {
		t.Fatalf("request span did not join trace: %+v", spans)
	}
}

// An unparseable traceparent is ignored: the request gets a fresh trace, no error.
func TestTraceIgnoresInvalidTraceparent(t *testing.T) {
	recorder := setupTracing(t)

	req := httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody)
	req.Header.Set("traceparent", "not-a-traceparent")
	rec := httptest.NewRecorder()
	Trace(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if spans := recorder.Ended(); len(spans) != 1 || !spans[0].SpanContext().IsValid() {
		t.Fatalf("expected one valid fresh span, got %+v", spans)
	}
}

// A 5xx is a failed request: the span must carry the error, not a success.
func TestTraceMarksServerError(t *testing.T) {
	recorder := setupTracing(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	Trace(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans=%d want 1", len(spans))
	}
	if spans[0].Status().Code != codes.Error || len(spans[0].Events()) == 0 {
		t.Fatalf("span status=%v events=%d, want an error", spans[0].Status(), len(spans[0].Events()))
	}
}

// A 4xx is an answered request, not a broken one: the span stays successful.
func TestTraceLeavesClientErrorSuccessful(t *testing.T) {
	recorder := setupTracing(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	Trace(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody))

	if spans := recorder.Ended(); len(spans) != 1 || spans[0].Status().Code == codes.Error {
		t.Fatalf("span status=%v want success", spans[0].Status())
	}
}

// A request refused before the router is still traced, with its 401 recorded.
func TestTraceSpansRejectedRequest(t *testing.T) {
	recorder := setupTracing(t)
	h := Trace(WithAuth(nil, okHandler(), discardLog()))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/me", http.NoBody))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
	if spans := recorder.Ended(); len(spans) != 1 || !spans[0].SpanContext().IsValid() {
		t.Fatalf("a rejected request must still produce a span, got %+v", spans)
	}
}
