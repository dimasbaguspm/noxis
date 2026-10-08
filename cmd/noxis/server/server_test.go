package server

import (
	"context"
	"testing"

	"noxis/internal/entity"
)

// TestNewVerifierOpenMode: an unset issuer must yield a nil interface, not a
// typed-nil *auth.OIDC (which would silently keep auth on).
func TestNewVerifierOpenMode(t *testing.T) {
	verifier, err := newVerifier(context.Background(), entity.Config{})
	if err != nil {
		t.Fatalf("open mode must not error: %v", err)
	}
	if verifier != nil {
		t.Fatalf("open mode verifier = %#v, want nil", verifier)
	}
}
