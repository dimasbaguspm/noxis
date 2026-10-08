package controller

import (
	"context"
	"net/http"

	"noxis/internal/entity"
)

type principalKey struct{}

// WithPrincipal carries the authenticated caller on the request context. The
// auth middleware (a different package) attaches it; handlers read it.
func WithPrincipal(ctx context.Context, p entity.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Principal returns the authenticated caller, or false when there is none.
func Principal(ctx context.Context) (entity.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(entity.Principal)
	return p, ok
}

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
func GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(_ context.Context, d Deps) (any, error) {
		return d.Auth, nil
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
func GetMe(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		p, ok := Principal(ctx)
		if !ok {
			return nil, errUnauthorized
		}
		levels, err := d.connectionLevels(ctx)
		if err != nil {
			return nil, err
		}
		return MeResponse{Principal: p, Connections: levels}, nil
	})
}

// connectionLevels is the effective level per connection for the caller. A
// global admin holds admin everywhere; everyone else only appears with a
// matching grant, so an absent connection means no access.
func (d Deps) connectionLevels(ctx context.Context) (map[string]string, error) {
	p, ok := Principal(ctx)
	if !ok {
		return map[string]string{}, nil
	}
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(list))
	for _, c := range list {
		if p.Admin {
			out[c.Name] = entity.LevelAdmin
			continue
		}
		level, err := d.effectiveLevel(ctx, c.Name, p.Groups)
		if err != nil {
			return nil, err
		}
		if level != "" {
			out[c.Name] = level
		}
	}
	return out, nil
}
