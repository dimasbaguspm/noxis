package api

import (
	"context"
	"net/http"

	"noxis/internal/usecases"
	"noxis/internal/entity"
)

// AuthInfo is the public OIDC configuration the console needs to start login.
// The issuer and client id are not secrets.
type AuthInfo struct {
	Enabled  bool   `json:"enabled"`
	Issuer   string `json:"issuer,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}

// @Summary Public OIDC configuration (console bootstrap)
// @Success 200 {object} AuthInfo
// @Router /auth/config [get]
func (s *Server) GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(context.Context) (any, error) {
		return s.Auth, nil
	})
}

// MeResponse is the caller plus the effective level it holds on each readable
// connection, so the console can gate actions without guessing.
type MeResponse struct {
	entity.Principal
	Connections map[string]string `json:"connections"`
}

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} MeResponse
// @Failure 401 {object} map[string]string
// @Router /v1/me [get]
func (s *Server) GetMe(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(ctx context.Context) (any, error) {
		p, ok := usecases.Principal(ctx)
		if !ok {
			return nil, errUnauthorized
		}
		levels, err := s.App.ConnectionLevels(ctx)
		if err != nil {
			return nil, err
		}
		return MeResponse{Principal: p, Connections: levels}, nil
	})
}