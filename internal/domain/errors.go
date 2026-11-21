// Package domain holds the entities, value objects, state machines and sentinel
// errors. It imports nothing outside the standard library: every other layer may
// depend on it, and it depends on none of them.
package domain

import (
	"errors"
	"fmt"
)

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

// NotAReference is the refusal a path param gets when it is not a ref. It says what a reference is,
// because the likeliest mistake is copying the NAME a list prints beside the reference it belongs to.
func NotAReference(value string) error {
	return fmt.Errorf("%w: %q is not a reference: a reference is the full UUID a list prints, e.g. 01a1fd59-6d1e-7e0d-8784-de451d9156c7", ErrInvalid, value)
}
