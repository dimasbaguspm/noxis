package proposal

import (
	"context"
	"errors"
	"noxis/internal/domain"
	"strings"
	"testing"
)

func TestCreateRehearsesAndJournals(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	f.driver.rehearsal = domain.Rehearsal{Status: domain.RehearsalVerified, SchemaDiff: "- a\n+ b"}

	out, err := NewCreate(f.deps).Run(context.Background(), CreateInput{
		TargetRef: f.target.Ref, Title: " add a column ", Query: "ALTER TABLE users ADD COLUMN note TEXT",
		Actor: "agent",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.Proposal.Kind != domain.KindDDL {
		t.Fatalf("kind = %s, want ddl", out.Proposal.Kind)
	}
	if out.Proposal.Title != "add a column" {
		t.Fatalf("title = %q, want it trimmed", out.Proposal.Title)
	}
	if out.Proposal.Status != domain.StatusVerified {
		t.Fatalf("status = %s, want verified after a conclusive rehearsal", out.Proposal.Status)
	}
	if out.Rehearsal.Ref == "" {
		t.Fatal("the rehearsal was returned without a reference")
	}
	if len(f.rehearsal.created) != 1 {
		t.Fatalf("stored %d rehearsals, want 1", len(f.rehearsal.created))
	}
	// The history is two lines: it was proposed, and it was rehearsed. Reading either one alone
	// hides half of what happened to it.
	kinds := []string{}
	for _, entry := range f.journal.entries {
		kinds = append(kinds, entry.Kind)
	}
	if len(kinds) != 2 || kinds[0] != "proposed" || kinds[1] != "rehearsed" {
		t.Fatalf("journal = %v, want [proposed rehearsed]", kinds)
	}
}

func TestCreateRefusals(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   CreateInput
		want error
	}{
		{
			"bad reference",
			CreateInput{TargetRef: "not-a-ref", Title: "t", Query: "ALTER TABLE u ADD COLUMN a INT"},
			domain.ErrInvalid,
		},
		{
			"bad tier",
			CreateInput{Title: "t", Query: "ALTER TABLE u ADD COLUMN a INT", Tier: "wishful"},
			domain.ErrInvalid,
		},
		{"a read is not a change", CreateInput{Title: "t", Query: "SELECT 1"}, domain.ErrInvalid},
		{"no title", CreateInput{Query: "ALTER TABLE u ADD COLUMN a INT"}, domain.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, domain.StatusDraft)
			if tc.in.TargetRef == "" {
				tc.in.TargetRef = f.target.Ref
			}
			if _, err := NewCreate(f.deps).Run(context.Background(), tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("Create: %v, want %v", err, tc.want)
			}
		})
	}
}

func TestListClampsPages(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	// Two rows and a limit of one: the page comes back with a cursor, and the extra row that was
	// fetched to decide that is not in it.
	f.proposals.stored["0192f0c0-0000-7000-8000-000000000003"] = domain.Proposal{
		Ref: "0192f0c0-0000-7000-8000-000000000003", Status: domain.StatusDraft,
	}

	out, err := NewList(f.deps).Run(context.Background(), ListInput{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out.Proposals) != 1 {
		t.Fatalf("page carried %d proposals, want 1", len(out.Proposals))
	}
	if out.NextCursor == "" {
		t.Fatal("a page with more rows did not name the cursor that follows")
	}

	for _, tc := range []struct {
		name string
		in   ListInput
	}{
		{"bad cursor", ListInput{Cursor: "nope"}},
		{"unknown status", ListInput{Status: "maybe"}},
		{"unknown kind", ListInput{Kind: "select"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewList(f.deps).Run(context.Background(), tc.in); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("List: %v, want ErrInvalid", err)
			}
		})
	}
}

func TestGetCarriesTheEvidence(t *testing.T) {
	f := newFixture(t, domain.StatusVerified)
	f.rehearsal.latest = domain.Rehearsal{Ref: "0192f0c0-0000-7000-8000-000000000004", Note: "shadow"}

	out, err := NewGet(f.deps).Run(context.Background(), f.proposal.Ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !out.HasRehearsal || out.Rehearsal.Note != "shadow" {
		t.Fatalf("Get = %+v, want the latest rehearsal", out)
	}

	// A proposal nothing could be attempted on has no rehearsal, and that is not an error.
	f.rehearsal.err = domain.ErrNotFound
	out, err = NewGet(f.deps).Run(context.Background(), f.proposal.Ref)
	if err != nil {
		t.Fatalf("Get without a rehearsal: %v", err)
	}
	if out.HasRehearsal {
		t.Fatal("Get invented a rehearsal")
	}

	if _, err := NewGet(f.deps).Run(context.Background(), "nope"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("Get with a bad reference: %v, want ErrInvalid", err)
	}
}

func TestRehearseRefusesADecidedProposal(t *testing.T) {
	f := newFixture(t, domain.StatusApplied)
	if _, _, err := NewRehearse(f.deps).Run(context.Background(), RehearseInput{Ref: f.proposal.Ref}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Rehearse on an applied proposal: %v, want ErrConflict", err)
	}

	f = newFixture(t, domain.StatusFailed)
	f.driver.rehearsal = domain.Rehearsal{Status: domain.RehearsalVerified}
	proposal, rehearsal, err := NewRehearse(f.deps).Run(context.Background(), RehearseInput{Ref: f.proposal.Ref, Tier: "static"})
	if err != nil {
		t.Fatalf("Rehearse: %v", err)
	}
	if proposal.Status != domain.StatusVerified || rehearsal.Status != domain.RehearsalVerified {
		t.Fatalf("rehearsed to %s/%s, want verified", proposal.Status, rehearsal.Status)
	}

	if _, _, err := NewRehearse(f.deps).Run(context.Background(),
		RehearseInput{Ref: f.proposal.Ref, Tier: "wishful"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("Rehearse at an unknown tier: %v, want ErrInvalid", err)
	}
}

// An engine that cannot be reached is stored as a failure, not raised as one: the attempt is a fact
// the reviewer needs to see.
func TestRehearseStoresAnUnreachableEngine(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	f.driver.rehearseErr = errors.New("dial tcp: connection refused")

	proposal, _, err := NewRehearse(f.deps).Run(context.Background(), RehearseInput{Ref: f.proposal.Ref})
	if err == nil {
		t.Fatal("Rehearse swallowed an unreachable engine")
	}
	if proposal.Status != domain.StatusFailed || proposal.Failure == "" {
		t.Fatalf("proposal = %+v, want failed with the cause on it", proposal)
	}
	if len(f.rehearsal.created) != 1 {
		t.Fatalf("the attempt was not stored: %d", len(f.rehearsal.created))
	}
}

func TestDecideSeparationOfDuties(t *testing.T) {
	f := newFixture(t, domain.StatusVerified)

	// The proposer cannot approve their own change.
	if _, err := NewDecide(f.deps).Run(context.Background(), DecideInput{
		Ref: f.proposal.Ref, Approve: true, Actor: "agent",
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("self-approval: %v, want ErrForbidden", err)
	}

	// A reviewer can, and the decision is journaled.
	approved, err := NewDecide(f.deps).Run(context.Background(), DecideInput{
		Ref: f.proposal.Ref, Approve: true, Actor: "reviewer", Note: "looks right",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if approved.Status != domain.StatusApproved || approved.DecidedBy != "reviewer" {
		t.Fatalf("decision = %+v", approved)
	}
	if len(f.journal.entries) != 1 || f.journal.entries[0].Kind != "approved" {
		t.Fatalf("journal = %+v, want one `approved` line", f.journal.entries)
	}
}

func TestDecideRefusals(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	if _, err := NewDecide(f.deps).Run(context.Background(), DecideInput{
		Ref: f.proposal.Ref, Approve: true, Actor: "reviewer",
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("deciding a draft: %v, want ErrConflict", err)
	}

	approved := newFixture(t, domain.StatusApproved)
	if _, err := NewDecide(approved.deps).Run(context.Background(), DecideInput{
		Ref: approved.proposal.Ref, Approve: true, Actor: "reviewer",
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("approving twice: %v, want ErrConflict", err)
	}

	if _, err := NewDecide(f.deps).Run(context.Background(), DecideInput{Ref: "nope", Approve: true}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("bad reference: %v, want ErrInvalid", err)
	}
}

// An admin approving their own proposal is allowed, and the journal is the record that says so.
func TestDecideAdminOverride(t *testing.T) {
	f := newFixture(t, domain.StatusVerified)
	proposal, err := NewDecide(f.deps).Run(context.Background(), DecideInput{
		Ref: f.proposal.Ref, Approve: true, Actor: "agent", ActorIsAdmin: true, Note: "no one else",
	})
	if err != nil {
		t.Fatalf("admin override: %v", err)
	}
	if proposal.Status != domain.StatusApproved {
		t.Fatalf("status = %s, want approved", proposal.Status)
	}
}

func TestApplyRecordsTheReceipt(t *testing.T) {
	f := newFixture(t, domain.StatusApproved)
	f.driver.receipt = domain.Receipt{Algorithm: domain.AlgorithmInstant, RowsAffected: 3}

	out, err := NewApply(f.deps).Run(context.Background(), ApplyInput{Ref: f.proposal.Ref, Actor: "ops"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Started {
		t.Fatal("a three-row change was sent to the background")
	}
	if out.Proposal.Status != domain.StatusApplied {
		t.Fatalf("status = %s, want applied", out.Proposal.Status)
	}
	if out.Proposal.Algorithm != domain.AlgorithmInstant || out.Proposal.RowsAffected != 3 {
		t.Fatalf("receipt = %+v", out.Proposal)
	}
	if len(f.driver.applied) != 1 {
		t.Fatalf("the statement ran %d times, want 1", len(f.driver.applied))
	}
	if len(f.journal.entries) != 1 || f.journal.entries[0].Detail["algorithm"] != "INSTANT" {
		t.Fatalf("journal = %+v, want the algorithm on the line", f.journal.entries)
	}
}

func TestApplyRefusals(t *testing.T) {
	if out, err := NewApply(newFixture(t, domain.StatusVerified).deps).Run(context.Background(),
		ApplyInput{Ref: "0192f0c0-0000-7000-8000-000000000002"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("applying an unapproved proposal: %v, want ErrConflict (out %+v)", err, out)
	}

	f := newFixture(t, domain.StatusApproved)
	f.proposal.Irreversible = true
	f.proposals.stored[f.proposal.Ref] = f.proposal
	if _, err := NewApply(f.deps).Run(context.Background(), ApplyInput{
		Ref: f.proposal.Ref,
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("destructive without acknowledgement: %v, want ErrInvalid", err)
	}

	// The claim is the lock: a proposal somebody else already took is a conflict, and the statement
	// never runs.
	claimed := newFixture(t, domain.StatusApproved)
	claimed.proposals.stored[claimed.proposal.Ref] = domain.Proposal{
		Ref: claimed.proposal.Ref, Status: domain.StatusApplying,
	}
	if _, err := NewApply(claimed.deps).Run(context.Background(), ApplyInput{
		Ref: claimed.proposal.Ref,
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("losing the claim: %v, want ErrConflict", err)
	}
	if len(claimed.driver.applied) != 0 {
		t.Fatal("the losing apply ran the statement")
	}
}

// A failed apply is stored, and the journal says so: a silent retry is how the same change lands
// twice.
func TestApplyStoresAFailure(t *testing.T) {
	f := newFixture(t, domain.StatusApproved)
	f.driver.applyErr = errors.New("Error 1064: you have an error in your SQL syntax")

	out, err := NewApply(f.deps).Run(context.Background(), ApplyInput{Ref: f.proposal.Ref, Actor: "ops"})
	if err == nil {
		t.Fatal("Apply swallowed the engine's refusal")
	}
	if out.Proposal.Status != domain.StatusFailed || !strings.Contains(out.Proposal.Failure, "1064") {
		t.Fatalf("proposal = %+v, want failed with the cause", out.Proposal)
	}
	if len(f.journal.entries) != 1 || f.journal.entries[0].Kind != "apply_failed" {
		t.Fatalf("journal = %+v, want one `apply_failed` line", f.journal.entries)
	}
}

// A statement too large to wait for answers at once and finishes behind the caller.
func TestApplyGoesBehindTheCallerOnALargeTable(t *testing.T) {
	f := newFixture(t, domain.StatusApproved)
	f.driver.plan = domain.Plan{RowsEstimated: 2_000_000}

	out, err := NewApply(f.deps).Run(context.Background(), ApplyInput{Ref: f.proposal.Ref, Actor: "ops"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.Started || out.Proposal.Status != domain.StatusApplying {
		t.Fatalf("out = %+v, want started and applying", out)
	}
	if out.Proposal.RowsEstimated != 2_000_000 {
		t.Fatalf("rows_estimated = %d", out.Proposal.RowsEstimated)
	}
}

func TestEventsRefusesAnUnknownProposal(t *testing.T) {
	f := newFixture(t, domain.StatusVerified)
	if _, err := NewEvents(f.deps).Run(context.Background(), f.proposal.Ref); err != nil {
		t.Fatalf("Events: %v", err)
	}
	if _, err := NewEvents(f.deps).Run(context.Background(), "0192f0c0-0000-7000-8000-0000000000ff"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Events for an unknown proposal: %v, want ErrNotFound", err)
	}
}
