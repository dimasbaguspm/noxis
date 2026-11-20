package handlers

import (
	"net/http"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/constants"
	"noxis/internal/domain"
)

type health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Healthz is public: a load balancer needs to know the process is up, and nothing more.
//
// @Summary Liveness
// @Description Answers without a token: the process is up and this is the version answering.
// @Tags system
// @Produce json
// @Success 200 {object} respond.Envelope
// @Router /healthz [get]
func (*Handlers) Healthz(w http.ResponseWriter, _ *http.Request) {
	respond.OK(w, health{Status: "ok", Version: constants.Version})
}

type whoami struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// Whoami answers the caller its own name and scopes, which is how a client checks a token
// before it does anything else.
// @Summary Who this token is
// @Description The identity and the scopes the API resolved for the bearer token.
// @Tags session
// @Produce json
// @Security BearerAuth
// @Success 200 {object} respond.Envelope
// @Failure 401 {object} respond.Envelope
// @Router /v1/whoami [get]
func (*Handlers) Whoami(w http.ResponseWriter, r *http.Request) {
	caller := middlewares.Caller(r.Context())
	respond.OK(w, whoami{Name: caller.Name, Scopes: scopes(caller)})
}

func scopes(caller domain.Caller) []string {
	if caller.Scopes == nil {
		return []string{}
	}
	return caller.Scopes
}
