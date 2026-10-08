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

// @Summary The authenticated caller
// @Success 200 {object} entity.Principal
// @Failure 401 {object} map[string]string
// @Router /v1/me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, _ Deps) (any, error) {
		p, ok := Principal(ctx)
		if !ok {
			return nil, errUnauthorized
		}
		return p, nil
	})
}
