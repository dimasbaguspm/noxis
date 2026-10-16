package auth

import "testing"

// TestPrincipalFromClaims locks the claim handling: a configurable claim name,
// non-string entries ignored, and admin only when the admin group is configured
// and present.
func TestPrincipalFromClaims(t *testing.T) {
	claims := map[string]any{
		"groups": []any{"dev", "noxis-admins", 42},
		"roles":  []any{"noxis-admins"},
	}
	p := principalFromClaims("alice", claims, "groups", "noxis-admins")
	if p.Subject != "alice" {
		t.Fatalf("subject=%q", p.Subject)
	}
	if len(p.Groups) != 2 || p.Groups[0] != "dev" || p.Groups[1] != "noxis-admins" {
		t.Fatalf("groups=%v (non-strings must be dropped)", p.Groups)
	}
	if !p.Admin {
		t.Fatal("admin group present but Admin is false")
	}

	custom := principalFromClaims("bob", claims, "roles", "noxis-admins")
	if len(custom.Groups) != 1 || !custom.Admin {
		t.Fatalf("custom claim not read: %+v", custom)
	}

	missing := principalFromClaims("carol", map[string]any{}, "groups", "noxis-admins")
	if missing.Admin || len(missing.Groups) != 0 {
		t.Fatalf("missing claim must mean no groups: %+v", missing)
	}

	noAdminGroup := principalFromClaims("dave", claims, "groups", "")
	if noAdminGroup.Admin {
		t.Fatal("Admin must be false when no admin group is configured")
	}
}
