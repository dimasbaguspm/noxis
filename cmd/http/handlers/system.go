package handlers

import (
	"net/http"

	"noxis/cmd/http/middlewares"
	"noxis/cmd/http/respond"
	"noxis/internal/constants"
	"noxis/internal/domain"
)

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

// Whoami answers the caller its own name and scopes, which is how a client checks a token
// before it does anything else.
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
