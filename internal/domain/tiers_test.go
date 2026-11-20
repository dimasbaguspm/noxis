package domain_test

import (
	"noxis/internal/constants"
	"noxis/internal/domain"
	"testing"
)

// The wire list of tiers lives in `constants` so a client can describe the API without importing the
// domain, and the typed list lives here because the state machine needs it. Two lists that could drift
// are one list with a test.
func TestConstantsTiersMatchTheDomain(t *testing.T) {
	if len(constants.Tiers) != len(domain.Tiers) {
		t.Fatalf("constants.Tiers = %v, domain.Tiers = %v", constants.Tiers, domain.Tiers)
	}
	for i, tier := range domain.Tiers {
		if constants.Tiers[i] != string(tier) {
			t.Fatalf("tier %d: constants %q, domain %q", i, constants.Tiers[i], tier)
		}
	}
}
