package release

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The published artifact is the contract: a commit, a release line, and nothing else needed to answer
// "am I behind?".
func TestLatestReadsThePublishedBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"version":"nightly","commit":"abc1234","built_at":"2026-11-21T10:00:00Z"}`))
	}))
	defer server.Close()

	info, err := NewChecker(server.URL+"/version.json", 0).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if info.Commit != "abc1234" || info.Version != "nightly" {
		t.Fatalf("info = %+v, want the published build", info)
	}
}

// One request per TTL, not one per caller: the console asks on every page load and the CLI on every
// command, and neither should cost a call to GitHub.
func TestLatestIsRememberedForTheTTL(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"version":"nightly","commit":"abc1234"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.URL, time.Hour)
	for range 5 {
		if _, err := checker.Latest(context.Background()); err != nil {
			t.Fatalf("Latest: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 within the TTL", calls)
	}
}

// A deployment with no outbound network answers with nothing rather than an error, and the answer is
// cached too: a dead URL must not be retried on every request.
func TestLatestSwallowsAFailedFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	info, err := NewChecker(server.URL, time.Hour).Latest(context.Background())
	if err == nil {
		t.Fatal("want the failure reported to the caller that logs it")
	}
	if info != nil {
		t.Fatalf("info = %+v, want nil", info)
	}
}

func TestNewerComparesCommits(t *testing.T) {
	for _, tc := range []struct {
		name      string
		published *Info
		commit    string
		want      bool
	}{
		{"a different commit is newer", &Info{Commit: "abc1234"}, "def5678", true},
		{"the same commit is not", &Info{Commit: "abc1234"}, "abc1234", false},
		{"nothing published is never newer", nil, "abc1234", false},
		{"an unstamped build cannot tell", &Info{Commit: "abc1234"}, "unknown", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Newer(tc.published, tc.commit); got != tc.want {
				t.Fatalf("Newer = %v, want %v", got, tc.want)
			}
		})
	}
}
