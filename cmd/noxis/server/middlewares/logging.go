package middlewares

import (
	"log/slog"
	"net/http"
	"time"

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

// RequestLog records method, path, status, duration and the caller of every
// request, and wraps the whole request in a span. It runs inside WithAuth, so
// the authenticated principal is on the context: its attributes are attached
// to the span once — every downstream span in the trace carries them without
// any deeper layer re-adding them.
func RequestLog(next http.Handler, log *slog.Logger) http.Handler {
	spanName := logging.CallerName(1)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, end := logging.StartSpan(r.Context(), spanName)
		logging.Attach(ctx, usecases.LogAttrs(ctx)...)
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r.WithContext(ctx))
		end(nil)
		log.InfoContext(ctx, "http",
			append([]any{
				"method", r.Method, "path", r.URL.Path,
				"status", rec.status, "dur", time.Since(start).String(),
			}, usecases.LogAttrs(ctx)...)...)
	})
}
