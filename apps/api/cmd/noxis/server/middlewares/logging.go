package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"noxis/internal/entity"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"noxis/internal/common/logging"
)

// statusRecorder captures the response status written by a handler, so the request log carries it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// statusCode is the status written, 200 when the handler never set one.
func (s *statusRecorder) statusCode() int {
	if s.status == 0 {
		return http.StatusOK
	}
	return s.status
}

// Trace opens a span before auth so rejected requests are traced and marks 5xx responses failed.
func Trace(next http.Handler) http.Handler {
	spanName := logging.CallerName(1)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// an upstream traceparent becomes the parent of this request's span
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, end := logging.StartSpan(ctx, spanName)
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r.WithContext(ctx))
		code := rec.statusCode()
		logging.Attach(ctx, "http.request.method", r.Method, "http.response.status_code", code)
		end(statusError(code))
	})
}

// statusError reports a server-side failure, so the span is not marked successful when the handler returned 5xx.
func statusError(code int) error {
	if code >= http.StatusInternalServerError {
		return fmt.Errorf("http %d", code)
	}
	return nil
}

// RequestLog records each routed request and attaches the authenticated caller to its span once.
func RequestLog(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		logging.Attach(ctx, entity.LogAttrs(ctx)...)
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		logging.Attach(ctx, "http.route", r.Pattern)
		log.InfoContext(ctx, "http",
			append([]any{
				"method", r.Method, "path", r.URL.Path, "route", r.Pattern,
				"status", rec.statusCode(), "dur", time.Since(start).String(),
			}, entity.LogAttrs(ctx)...)...)
	})
}
