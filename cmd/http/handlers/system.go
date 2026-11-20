// Package handlers is the HTTP transport. A handler reads a request, calls one use case
// and writes an envelope: no SQL, no policy and no business rule lives here.
package handlers

import (
	"log/slog"
	"net/http"

	"noxis/cmd/http/middlewares"
	"noxis/cmd/http/respond"
	"noxis/internal/constants"
	"noxis/internal/domain"
)

// Handlers carries what a handler may use. Dependencies arrive as fields as they gain a
// use case; nothing is reached for from a package-level variable.
type Handlers struct {
	log *slog.Logger
}

// New builds the handler set.
func New(log *slog.Logger) *Handlers { return &Handlers{log: log} }

type health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Healthz is public: a load balancer needs to know the process is up, and nothing more.
func (h *Handlers) Healthz(w http.ResponseWriter, _ *http.Request) {
	respond.OK(w, health{Status: "ok", Version: constants.Version})
}

type whoami struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// Whoami answers the caller its own name and scopes, which is how a client checks a
// token before it does anything else.
func (h *Handlers) Whoami(w http.ResponseWriter, r *http.Request) {
	caller := middlewares.Caller(r.Context())
	respond.OK(w, whoami{Name: caller.Name, Scopes: scopes(caller)})
}

func scopes(caller domain.Caller) []string {
	if caller.Scopes == nil {
		return []string{}
	}
	return caller.Scopes
}
