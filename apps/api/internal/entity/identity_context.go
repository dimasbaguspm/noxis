package entity

import "context"

// The authenticated caller travels on the request context. It sits here, with
// the identity it carries, so every layer above — a use case, a middleware, a
// test helper — reads it the same way.

type principalKey struct{}

// WithPrincipal carries the authenticated caller on the request context. The
// contract: the auth middleware sets the principal once per request; use cases
// read it (actor, LogAttrs) for authorization and attribution; modules and
// repositories never read it — policy is applied before they run.
func WithPrincipal(ctx context.Context, p Identity) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Principal returns the authenticated caller, or false when there is none.
func Principal(ctx context.Context) (Identity, bool) {
	p, ok := ctx.Value(principalKey{}).(Identity)
	return p, ok
}

// LogAttrs renders the authenticated caller for structured logs: the subject as
// user_id, the role and the full group list. No principal returns nil so logs
// stay unattributed rather than empty.
func LogAttrs(ctx context.Context) []any {
	p, ok := Principal(ctx)
	if !ok {
		return nil
	}
	return []any{"user_id", p.Subject, "role", p.Role, "groups", p.Groups}
}

// Actor returns the caller's subject for attribution; empty when there is no
// principal.
func Actor(ctx context.Context) string {
	if p, ok := Principal(ctx); ok {
		return p.Subject
	}
	return ""
}
