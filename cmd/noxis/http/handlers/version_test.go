package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/cmd/noxis/http/handlers"
	"noxis/internal/common/release"
	"noxis/internal/constants"
	"testing"
)

// fixedRelease answers the one question the version route asks, without a network.
type fixedRelease struct {
	info *release.Info
	err  error
}

func (f fixedRelease) Latest(context.Context) (*release.Info, error) { return f.info, f.err }

func versionRoute(t *testing.T, checker handlers.ReleaseChecker) map[string]any {
	t.Helper()
	h := handlers.New(handlers.Deps{
		Log:     slog.New(slog.DiscardHandler),
		Release: checker,
	})
	rec := httptest.NewRecorder()
	h.Version(rec, httptest.NewRequest(http.MethodGet, "/v1/version", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("the answer is not json: %v", err)
	}
	return body.Data
}

// The route names the build, the page every notice points at, and — when the published build could be
// read — whether it is a different commit than this one.
func TestVersionNamesTheBuildAndTheNotice(t *testing.T) {
	data := versionRoute(t, fixedRelease{info: &release.Info{Version: "nightly", Commit: "abc1234"}})

	if got := data["release_url"]; got != constants.ReleaseURL {
		t.Fatalf("release_url = %v, want %q", got, constants.ReleaseURL)
	}
	latest, ok := data["latest"].(map[string]any)
	if !ok || latest["commit"] != "abc1234" {
		t.Fatalf("latest = %v, want the published commit", data["latest"])
	}
	// This binary is unstamped in a unit test, so a published commit is not "newer" than an unknown one:
	// a notice that fires on every run is worse than none.
	if data["newer"] != false {
		t.Fatalf("newer = %v, want false for an unstamped build", data["newer"])
	}
}

// A deployment that cannot reach GitHub still answers: `latest` is null and nothing else changes.
func TestVersionSurvivesAnUnreachableReleasePage(t *testing.T) {
	data := versionRoute(t, fixedRelease{err: errors.New("dial tcp: no route to host")})

	if data["latest"] != nil {
		t.Fatalf("latest = %v, want null", data["latest"])
	}
	if data["newer"] != false {
		t.Fatalf("newer = %v, want false", data["newer"])
	}
	if data["version"] == nil || data["version"] == "" {
		t.Fatal("version is empty: the route must always say what it is")
	}
}
