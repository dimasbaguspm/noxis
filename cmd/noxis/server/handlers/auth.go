package handlers

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// @Summary Public OIDC configuration (console bootstrap)
// @Success 200 {object} config.AuthInfo
// @Router /auth/config [get]
func GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(context.Context) (any, error) {
		return d.Auth, nil
	})
}

// MeResponse is the caller plus the effective level it holds on each readable
// connection, so the console can gate actions without guessing.
type MeResponse struct {
	entity.Identity
	Connections map[string]string `json:"connections"`
}

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} MeResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /v1/me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		p, ok := usecases.Principal(ctx)
		if !ok {
			return nil, errUnauthorized
		}
		levels, err := d.Me.ConnectionLevels(ctx)
		if err != nil {
			return nil, err
		}
		return MeResponse{Identity: p, Connections: levels}, nil
	})
}
