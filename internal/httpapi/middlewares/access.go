package middlewares

import (
	"log/slog"
	"net/http"
	"noxis/internal/common/ref"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/httpapi/respond"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// Chain wraps a handler with the given middlewares, outermost first.
func Chain(next http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		next = mw[i](next)
	}
	return next
}

// Access logs one line per request and mints the request id when the caller sent none,
// so even a refused request carries one back. It also opens the root span, so every
// span a handler or an adapter records hangs under this request.
func Access(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			id := r.Header.Get(constants.HeaderRequestID)
			if id == "" {
				id = ref.New()
			}
			w.Header().Set(constants.HeaderRequestID, id)
			ctx, end := telemetry.Span(r.Context(), "http request",
				attribute.String("request.id", id), attribute.String("http.route", r.URL.Path))
			rec := &recorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r.WithContext(ctx))
			end(nil)
			log.InfoContext(ctx, "request",
				"id", id,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"ms", time.Since(started).Milliseconds(),
			)
		})
	}
}

// Span opens one span per route, named after the package, type and method that serves it. It sits
// inside the request span, so a trace reads `http POST /proposals/{ref}/apply` then
// `noXis/handlers.ApplyProposal` then whatever the engine recorded underneath.
func Span(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, end := telemetry.Span(r.Context(), name)
			defer end(nil)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Recover turns a panic into a logged cause and a 500 the client can read.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic", "path", r.URL.Path, "panic", v)
					respond.Refuse(w, http.StatusInternalServerError, constants.CodeInternal,
						"the server could not complete the request")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// recorder remembers the status the handler chose, because http.ResponseWriter does not.
type recorder struct {
	http.ResponseWriter
	status int
}

func (r *recorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
