package commands

import (
	"noxis/internal/client"
	"testing"
)

// A link is what an agent quotes and a person opens, so its shape is fixed here: the console's own route,
// the target first and the proposal second.
func TestLinkForIsTheConsolesOwnRoute(t *testing.T) {
	proposal := client.Proposal{
		Ref:       "01a1fe7d-9de3-7016-9a4b-3c2d1e0f9a8b",
		TargetRef: "01a1fe92-1111-7222-8333-444455556666",
	}
	got := linkFor("https://noxis.example.com", proposal)
	want := "https://noxis.example.com/targets/01a1fe92-1111-7222-8333-444455556666/proposals/01a1fe7d-9de3-7016-9a4b-3c2d1e0f9a8b"
	if got != want {
		t.Fatalf("linkFor = %q, want %q", got, want)
	}
	// A deployment mounted under a path keeps it: PUBLIC_URL is an origin plus a prefix, and the route is
	// appended to whatever it said.
	if got := linkFor("https://example.com/noxis", proposal); got != "https://example.com/noxis/targets/01a1fe92-1111-7222-8333-444455556666/proposals/01a1fe7d-9de3-7016-9a4b-3c2d1e0f9a8b" {
		t.Fatalf("prefixed linkFor = %q", got)
	}
}

// No origin, or a proposal without the two refs the route needs, is no link. A guessed URL would send a
// reviewer to the wrong deployment, which is worse than a ref they have to paste.
func TestLinkForRefusesWhatItCannotBuild(t *testing.T) {
	full := client.Proposal{Ref: "a", TargetRef: "b"}
	for _, tc := range []struct {
		name     string
		origin   string
		proposal client.Proposal
	}{
		{"no origin", "", full},
		{"no target", "https://noxis.example.com", client.Proposal{Ref: "a"}},
		{"no proposal", "https://noxis.example.com", client.Proposal{TargetRef: "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := linkFor(tc.origin, tc.proposal); got != "" {
				t.Fatalf("linkFor = %q, want nothing", got)
			}
		})
	}
}
