// Package testkit is the two helpers a use-case test needs to drive a Serve method: a mux that fills the path values the router fills, and one request.
package testkit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"noxis/internal/entity"
)

// Mux registers one Serve on a fresh mux, so {engine}, {name} and {id} are set the way the real router sets them.
func Mux(method, pattern string, h http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, h)
	return mux
}

// Do runs one request against a handler and returns the recorder.
func Do(h http.Handler, method, path, body string, p *entity.Identity) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if p != nil {
		req = req.WithContext(entity.WithPrincipal(req.Context(), *p))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
