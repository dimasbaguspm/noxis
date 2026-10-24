package entity

import "context"

// The request principal lives here so every layer reads it without importing usecases.

type principalKey struct{}

// WithPrincipal carries the authenticated caller on the request context.
func WithPrincipal(ctx context.Context, p Identity) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Principal returns the authenticated caller, or false when there is none.
func Principal(ctx context.Context) (Identity, bool) {
	p, ok := ctx.Value(principalKey{}).(Identity)
	return p, ok
}

// LogAttrs returns the caller's id, role, and groups for structured logs; no principal yields no attributes.
func LogAttrs(ctx context.Context) []any {
	p, ok := Principal(ctx)
	if !ok {
		return nil
	}
	return []any{"user_id", p.Subject, "user_role", p.Role, "groups", p.Groups}
}

// Actor returns the caller's subject for attribution; empty when there is no principal.
func Actor(ctx context.Context) string {
	if p, ok := Principal(ctx); ok {
		return p.Subject
	}
	return ""
}
