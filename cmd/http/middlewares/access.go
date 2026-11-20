package middlewares

import (
	"log/slog"
	"net/http"
	"time"

	"noxis/cmd/http/respond"
	"noxis/internal/common/ref"
	"noxis/internal/constants"
)

// Chain wraps a handler with the given middlewares, outermost first.
func Chain(next http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		next = mw[i](next)
	}
	return next
}

// Access logs one line per request and mints the request id when the caller sent none,
// so even a refused request carries one back.
func Access(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			id := r.Header.Get(constants.HeaderRequestID)
			if id == "" {
				id = ref.New()
			}
			w.Header().Set(constants.HeaderRequestID, id)
			rec := &recorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("request",
				"id", id,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"ms", time.Since(started).Milliseconds(),
			)
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
