package domain_test

import (
	"errors"
	"noxis/internal/domain"
	"strings"
	"testing"
)

// A proposal a reviewer cannot act on is refused before it is stored, and every refusal says why.
func TestProposalValidate(t *testing.T) {
	full := domain.Proposal{
		Ref: "p", TargetRef: "t", Title: "add a column", Kind: domain.KindDDL,
		Query: "ALTER TABLE users ADD COLUMN note TEXT", Tier: domain.TierShadow,
		ProposedBy: "agent",
	}

	for _, tc := range []struct {
		name   string
		change func(*domain.Proposal)
		want   string
	}{
		{"ok", func(*domain.Proposal) {}, ""},
		{"no title", func(p *domain.Proposal) { p.Title = "   " }, "title"},
		{"long title", func(p *domain.Proposal) { p.Title = strings.Repeat("a", 201) }, "200"},
		{"no statement", func(p *domain.Proposal) { p.Query = " " }, "statement"},
		{"bad kind", func(p *domain.Proposal) { p.Kind = "select" }, "neither dml nor ddl"},
		{"bad tier", func(p *domain.Proposal) { p.Tier = "wishful" }, "tier"},
		{"no author", func(p *domain.Proposal) { p.ProposedBy = "" }, "caller"},
		{"no target", func(p *domain.Proposal) { p.TargetRef = "" }, "target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A copy per case: a table of cases that share one value is a table that tests the last case.
			subject := full
			tc.change(&subject)
			err := subject.Validate()
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("Validate: %v, want nil", err)
			case tc.want != "" && err == nil:
				t.Fatalf("Validate accepted %+v", subject)
			case tc.want != "" && !errors.Is(err, domain.ErrInvalid):
				t.Fatalf("Validate: %v, want ErrInvalid", err)
			case tc.want != "" && !strings.Contains(err.Error(), tc.want):
				t.Fatalf("Validate: %q does not mention %q", err, tc.want)
			}
		})
	}
}

// The state machine is the only writer of status, so what it allows is the whole policy. Anything
// not listed has to be refused, not silently allowed.
func TestProposalTransitions(t *testing.T) {
	allowed := map[domain.Status][]domain.Status{
		domain.StatusDraft:    {domain.StatusVerified, domain.StatusFailed},
		domain.StatusFailed:   {domain.StatusDraft, domain.StatusVerified},
		domain.StatusVerified: {domain.StatusApproved, domain.StatusRejected, domain.StatusFailed},
		domain.StatusApproved: {domain.StatusApplying, domain.StatusRejected},
		domain.StatusApplying: {domain.StatusApplied, domain.StatusFailed},
		domain.StatusApplied:  {},
		domain.StatusRejected: {},
	}
	all := []domain.Status{
		domain.StatusDraft, domain.StatusVerified, domain.StatusFailed, domain.StatusApproved,
		domain.StatusRejected, domain.StatusApplying, domain.StatusApplied,
	}

	for from, targets := range allowed {
		for _, to := range all {
			want := false
			for _, candidate := range targets {
				if candidate == to {
					want = true
				}
			}
			proposal := domain.Proposal{Status: from}
			if got := proposal.CanTransition(to); got != want {
				t.Fatalf("%s -> %s = %v, want %v", from, to, got, want)
			}
		}
	}
}

// Approving twice is not allowed, and applying something unapproved is the mistake the claim exists
// to prevent: both are predicates the transport reads, so both are pinned here.
func TestProposalPredicates(t *testing.T) {
	for _, tc := range []struct {
		status    domain.Status
		decidable bool
		applyable bool
	}{
		{domain.StatusDraft, false, false},
		{domain.StatusVerified, true, false},
		{domain.StatusApproved, true, true},
		{domain.StatusApplying, false, false},
		{domain.StatusApplied, false, false},
		{domain.StatusRejected, false, false},
	} {
		proposal := domain.Proposal{Status: tc.status}
		if got := proposal.Decidable(); got != tc.decidable {
			t.Fatalf("%s Decidable = %v, want %v", tc.status, got, tc.decidable)
		}
		if got := proposal.Applyable(); got != tc.applyable {
			t.Fatalf("%s Applyable = %v, want %v", tc.status, got, tc.applyable)
		}
	}
}

func TestIsTier(t *testing.T) {
	for _, tier := range domain.Tiers {
		if !domain.IsTier(string(tier)) {
			t.Fatalf("IsTier(%q) = false", tier)
		}
	}
	if domain.IsTier("thorough") {
		t.Fatal("IsTier accepted a tier this build does not run")
	}
}

// A copy is the algorithm that blocks writes; the others do not, and a receipt that says COPY is a
// warning the reviewer has to be able to trust.
func TestAlgorithmLocked(t *testing.T) {
	if !domain.AlgorithmCopy.Locked() {
		t.Fatal("COPY is not reported as locking the table")
	}
	for _, algorithm := range []domain.Algorithm{domain.AlgorithmInstant, domain.AlgorithmInPlace, ""} {
		if algorithm.Locked() {
			t.Fatalf("%q reported as locking the table", algorithm)
		}
	}
}

// A partial rehearsal is approvable — that is how a destructive change reaches an operator — and a
// failed one is not. The distinction is what the reviewer reads the note for.
func TestRehearsalVerdicts(t *testing.T) {
	for _, tc := range []struct {
		status     domain.RehearsalStatus
		conclusive bool
		usable     bool
	}{
		{domain.RehearsalVerified, true, true},
		{domain.RehearsalPartial, false, true},
		{domain.RehearsalFailed, false, false},
		{domain.RehearsalUnsupported, false, false},
	} {
		rehearsal := domain.Rehearsal{Status: tc.status}
		if got := rehearsal.Conclusive(); got != tc.conclusive {
			t.Fatalf("%s Conclusive = %v, want %v", tc.status, got, tc.conclusive)
		}
		if got := rehearsal.Usable(); got != tc.usable {
			t.Fatalf("%s Usable = %v, want %v", tc.status, got, tc.usable)
		}
	}
}

func TestCallerScopes(t *testing.T) {
	caller := domain.Caller{Name: "agent", Scopes: []string{"read", "propose"}}
	if !caller.Has("read") {
		t.Fatal("Has(read) = false")
	}
	if caller.Has("apply") {
		t.Fatal("Has(apply) = true for a caller that does not hold it")
	}
	if !caller.HasAll([]string{"read", "propose"}) {
		t.Fatal("HasAll(read, propose) = false")
	}
	if caller.HasAll([]string{"read", "apply"}) {
		t.Fatal("HasAll(read, apply) = true with no apply scope")
	}
	if !caller.HasAll(nil) {
		t.Fatal("HasAll(nil) = false: a route with no scope must not require one")
	}
	if domain.Anonymous.Name != "" || domain.Anonymous.Has("read") {
		t.Fatal("Anonymous holds a scope")
	}
}

// Progress that is not running says so with -1 rather than 0: a zero percent is a statement that has
// started, and an unknown one is not the same fact.
func TestUnknownProgress(t *testing.T) {
	if domain.UnknownProgress.Percent != -1 || domain.UnknownProgress.Stage != "" {
		t.Fatalf("UnknownProgress = %+v", domain.UnknownProgress)
	}
	if domain.UnknownProgress.Percent == 0 {
		t.Fatal("unknown progress reported as zero percent")
	}
}
