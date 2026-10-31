package auth

import "testing"

// TestPrincipalFromClaims locks the claim handling: a configurable claim name, non-string entries ignored, and no global role from the issuer.
func TestPrincipalFromClaims(t *testing.T) {
	claims := map[string]any{
		"groups": []any{"dev", "noxis-admins", 42},
		"roles":  []any{"noxis-admins"},
	}
	p := principalFromClaims("alice", claims, "groups")
	if p.Subject != "alice" {
		t.Fatalf("subject=%q", p.Subject)
	}
	if len(p.Groups) != 2 || p.Groups[0] != "dev" || p.Groups[1] != "noxis-admins" {
		t.Fatalf("groups=%v (non-strings must be dropped)", p.Groups)
	}
	if p.IsSuper() {
		t.Fatal("an OIDC claim must never grant the super role")
	}

	custom := principalFromClaims("bob", claims, "roles")
	if len(custom.Groups) != 1 {
		t.Fatalf("custom claim not read: %+v", custom)
	}

	missing := principalFromClaims("carol", map[string]any{}, "groups")
	if len(missing.Groups) != 0 {
		t.Fatalf("missing claim must mean no groups: %+v", missing)
	}
}
