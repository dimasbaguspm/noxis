// Package domain holds the entities, value objects, state machines and sentinel
// errors. It imports nothing outside the standard library: every other layer may
// depend on it, and it depends on none of them.
package domain

import "errors"

// Sentinel errors. Every layer wraps these with %w and the transport compares them
// with errors.Is, so a new error never needs a new status mapping.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrUnsupported  = errors.New("unsupported")
	ErrInvalid      = errors.New("invalid")
)
