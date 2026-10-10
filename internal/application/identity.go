package application

import (
	"context"

	"noxis/internal/entity"
)

type principalKey struct{}

// WithPrincipal carries the authenticated caller on the request context. The
// auth middleware (in cmd/noxis/server) attaches it; workflows read it.
func WithPrincipal(ctx context.Context, p entity.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Principal returns the authenticated caller, or false when there is none.
func Principal(ctx context.Context) (entity.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(entity.Principal)
	return p, ok
}

// actor returns the caller's subject for attribution; empty in open mode.
func actor(ctx context.Context) string {
	if p, ok := Principal(ctx); ok {
		return p.Subject
	}
	return ""
}

// ConnectionLevels is the effective level per connection for the caller. A
// global admin holds admin everywhere; everyone else only appears with a
// matching grant, so an absent connection means no access.
func (a *App) ConnectionLevels(ctx context.Context) (map[string]string, error) {
	p, ok := Principal(ctx)
	if !ok {
		return map[string]string{}, nil
	}
	list, err := a.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(list))
	for i := range list {
		c := &list[i]
		if p.Admin {
			out[c.Name] = entity.LevelAdmin
			continue
		}
		level, err := a.EffectiveLevel(ctx, c.Name, p.Groups)
		if err != nil {
			return nil, err
		}
		if level != "" {
			out[c.Name] = level
		}
	}
	return out, nil
}