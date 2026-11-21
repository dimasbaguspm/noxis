package handlers

import (
	"net/http"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/common/release"
	"noxis/internal/constants"
	"noxis/internal/domain"
)

type Health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Healthz is public: a load balancer needs to know the process is up, and nothing more.
//
// @Summary Liveness
// @Description Answers without a token: the process is up and this is the version answering.
// @Tags system
// @Produce json
// @Success 200 {object} Health
// @Router /healthz [get]
func (*Handlers) Healthz(w http.ResponseWriter, _ *http.Request) {
	respond.OK(w, Health{Status: "ok", Version: constants.Build()})
}

// VersionInfo is what this binary is, and whether a newer build was published. `latest` is null when the
// published build could not be read — an update check that fails is a missing notice, not a failed request.
type VersionInfo struct {
	Version    string        `json:"version"`
	Commit     string        `json:"commit"`
	ReleaseURL string        `json:"release_url"`
	Latest     *release.Info `json:"latest"`
	Newer      bool          `json:"newer"`
}

// Version is public: the version of the binary answering, the page every update notice points at, and the
// newest build the publish job uploaded when this deployment could read it (checked at most hourly).
//
// @Summary What this build is, and whether there is a newer one
// @Description The version answering, the release page, and the newest published build when it could be read.
// @Tags system
// @Produce json
// @Success 200 {object} VersionInfo
// @Router /v1/version [get]
func (h *Handlers) Version(w http.ResponseWriter, r *http.Request) {
	var published *release.Info
	if h.Release != nil {
		latest, err := h.Release.Latest(r.Context())
		if err != nil {
			// Debug, not error: a deployment with no outbound network is a deployment, not a fault.
			h.Log.Debug("the published build could not be read", "error", err)
		}
		published = latest
	}
	respond.OK(w, VersionInfo{
		Version:    constants.Build(),
		Commit:     constants.Commit,
		ReleaseURL: constants.ReleaseURL,
		Latest:     published,
		Newer:      release.Newer(published, constants.Commit),
	})
}

type Whoami struct {
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
// @Success 200 {object} Whoami
// @Failure 401 {object} respond.Envelope
// @Router /v1/whoami [get]
func (*Handlers) Whoami(w http.ResponseWriter, r *http.Request) {
	caller := middlewares.Caller(r.Context())
	respond.OK(w, Whoami{Name: caller.Name, Scopes: scopes(caller)})
}

func scopes(caller domain.Caller) []string {
	if caller.Scopes == nil {
		return []string{}
	}
	return caller.Scopes
}
