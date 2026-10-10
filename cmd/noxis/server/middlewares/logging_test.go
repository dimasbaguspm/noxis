package middlewares

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	req := httptest.NewRequest(http.MethodGet, "/v1/engine/sql/connections", http.NoBody)
	RequestLog(inner, log).ServeHTTP(rec, req)

	line := buf.String()
	for _, want := range []string{`"method":"GET"`, `"path":"/v1/engine/sql/connections"`, `"status":404`} {
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
	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Principal{Subject: "bob", Groups: []string{"readers"}}))
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
	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	req = req.WithContext(usecases.WithPrincipal(req.Context(), entity.Principal{Subject: "bob"}))
	RequestLog(okHandler(), log).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}
