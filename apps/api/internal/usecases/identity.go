package usecases

import (
	"context"

	"noxis/internal/entity"
)

type principalKey struct{}

// WithPrincipal carries the authenticated caller on the request context. The
// contract: the auth middleware sets the principal once per request; usecases
// read it (actor, LogAttrs) for authorization and attribution; modules and
// repositories never read it — policy is applied before they run.
func WithPrincipal(ctx context.Context, p entity.Identity) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Principal returns the authenticated caller, or false when there is none.
func Principal(ctx context.Context) (entity.Identity, bool) {
	p, ok := ctx.Value(principalKey{}).(entity.Identity)
	return p, ok
}

// LogAttrs renders the authenticated caller for structured logs: the subject
// as user_id, the admin flag and the full group list. Open mode (no principal)
// returns nil so logs stay unattributed rather than empty.
func LogAttrs(ctx context.Context) []any {
	p, ok := Principal(ctx)
	if !ok {
		return nil
	}
	return []any{"user_id", p.Subject, "super", p.Super, "groups", p.Groups}
}

// Actor returns the caller's subject for attribution; empty when there is no
// principal.
func Actor(ctx context.Context) string {
	if p, ok := Principal(ctx); ok {
		return p.Subject
	}
	return ""
}
