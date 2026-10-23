package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"noxis/internal/common/logging"
	"noxis/internal/usecases"
)

// statusRecorder captures the response status written by a handler, so the
// request log carries it.
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

// Trace opens the request's span and records its outcome. It runs outside
// WithAuth, so a request refused before the router — a missing or invalid
// token — still has a trace it can be found by, and it closes the span with an
// error when the handler answered 5xx, so a failed request is not recorded as a
// success.
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

// statusError reports a server-side failure, so the span is not marked successful
// when the handler returned 5xx. A client error is a answered request, not a
// broken one.
func statusError(code int) error {
	if code >= http.StatusInternalServerError {
		return fmt.Errorf("http %d", code)
	}
	return nil
}

// RequestLog records method, path, route, status, duration and the caller of
// every request that reached the router. It runs inside WithAuth, so the
// authenticated principal is on the context: its attributes are attached to
// the request span once — every downstream span in the trace carries them
// without any deeper layer re-adding them.
func RequestLog(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		logging.Attach(ctx, usecases.LogAttrs(ctx)...)
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		logging.Attach(ctx, "http.route", r.Pattern)
		log.InfoContext(ctx, "http",
			append([]any{
				"method", r.Method, "path", r.URL.Path, "route", r.Pattern,
				"status", rec.statusCode(), "dur", time.Since(start).String(),
			}, usecases.LogAttrs(ctx)...)...)
	})
}
