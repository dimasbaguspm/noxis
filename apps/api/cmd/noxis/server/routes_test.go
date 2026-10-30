package server

import (
	"net/http"
	"strings"
	"testing"

	"noxis/internal/app"
	"noxis/internal/common/config"
	"noxis/internal/constants"
)

// TestRoutesAreUnique rejects two registrations of the same method and path:
// Go's mux would panic at startup, but the table should fail in a unit test.
func TestRoutesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, rt := range routes(app.App{}, config.Config{}) {
		key := rt.Method + " " + rt.Endpoint
		if seen[key] {
			t.Fatalf("duplicate route %s", key)
		}
		seen[key] = true
		if rt.Handler == nil {
			t.Fatalf("route %s has no handler", key)
		}
	}
}

// TestOnlyLoginAndConfigArePublic locks the access rule to the two routes the
// console calls before it has a token; any other public API route is a leak.
func TestOnlyLoginAndConfigArePublic(t *testing.T) {
	want := map[string]bool{
		http.MethodGet + " " + constants.APIPrefix + "/auth/config": true,
		http.MethodPost + " " + constants.APIPrefix + "/auth/login": true,
	}
	got := map[string]bool{}
	for _, rt := range routes(app.App{}, config.Config{}) {
		if !strings.HasPrefix(rt.Endpoint, constants.APIPrefix) || !rt.Public {
			continue
		}
		key := rt.Method + " " + rt.Endpoint
		got[key] = true
		if !want[key] {
			t.Errorf("unexpected public route %s", key)
		}
	}
	for key := range want {
		if !got[key] {
			t.Errorf("route %s is no longer public", key)
		}
	}
}
