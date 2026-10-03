package controller

import (
	_ "embed"
	"net/http"
)

// The operator console: a single-page, dependency-free HTML app served at /
// and driven entirely by this API (same origin). No build step — edit
// index.html and rebuild the binary.
//
//go:embed index.html
var consoleHTML []byte

// GetConsole serves the operator UI (status, spec assignment, doorman ops).
func GetConsole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(consoleHTML)
}
