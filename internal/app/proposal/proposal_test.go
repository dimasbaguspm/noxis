package proposal

import (
	"context"
	"errors"
	"noxis/internal/common/keyed"
	"noxis/internal/domain"
	"strings"
	"testing"
	"time"
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
	applied := journalEntry(f.journal.entries, "applied")
	if applied == nil || applied.Detail["algorithm"] != "INSTANT" {
		t.Fatalf("journal = %+v, want the algorithm on the applied line", f.journal.entries)
	}
	// The change landed, so the engine is asked again: the instance page describes the server the change
	// ran against, not the one the registration happened to meet.
	if probed := journalEntry(f.journal.entries, "probed"); probed == nil {
		t.Fatalf("journal = %+v, want the target probed after the change", f.journal.entries)
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

// A proposal retried under the key it named the first time is one change, with the evidence the first
// call produced: nothing is rehearsed twice and nothing is journaled twice.
func TestCreateWithTheSameKeyIsOneProposal(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	f.driver.rehearsal = domain.Rehearsal{Status: domain.RehearsalVerified, SchemaDiff: "- a\\n+ b"}
	f.rehearsal.latest = domain.Rehearsal{Status: domain.RehearsalVerified, SchemaDiff: "- a\\n+ b"}
	in := CreateInput{
		TargetRef: f.target.Ref, Title: "add a column", Query: "ALTER TABLE users ADD COLUMN note TEXT",
		Tier: "shadow", Actor: "agent", IdempotencyKey: "retry-me",
	}

	first, err := NewCreate(f.deps).Run(context.Background(), in)
	if err != nil {
		t.Fatalf("first propose: %v", err)
	}
	second, err := NewCreate(f.deps).Run(context.Background(), in)
	if err != nil {
		t.Fatalf("retried propose: %v", err)
	}

	if second.Proposal.Ref != first.Proposal.Ref {
		t.Fatalf("refs = %q and %q, want the same proposal", first.Proposal.Ref, second.Proposal.Ref)
	}
	if first.Replayed || !second.Replayed {
		t.Fatalf("replayed = %v then %v, want false then true", first.Replayed, second.Replayed)
	}
	// The fixture seeds one proposal of its own, so the count that matters is the one carrying the key.
	keyed := 0
	for _, proposal := range f.proposals.stored {
		if proposal.IdempotencyKey == in.IdempotencyKey {
			keyed++
		}
	}
	if keyed != 1 {
		t.Fatalf("stored %d proposals under the key, want 1", keyed)
	}
	if second.Rehearsal.Status != domain.RehearsalVerified {
		t.Fatalf("the replay answered with no evidence: %+v", second.Rehearsal)
	}
}

// A rehearsal can prove less than it was asked to: a server that will not give the target a throwaway
// database answers a static check instead. The proposal records the tier that ran, so neither the
// queue nor the page can claim a shadow run that never happened.
func TestRehearseRecordsTheTierThatRan(t *testing.T) {
	f := newFixture(t, domain.StatusVerified)
	f.driver.rehearsal = domain.Rehearsal{
		Status: domain.RehearsalPartial,
		Tier:   domain.TierStatic,
		Note:   "static: the statement was parsed and planned",
	}

	out, rehearsal, err := NewRehearse(f.deps).Run(context.Background(), RehearseInput{Ref: f.proposal.Ref})
	if err != nil {
		t.Fatalf("Rehearse: %v", err)
	}
	if rehearsal.Tier != domain.TierStatic {
		t.Fatalf("rehearsal tier = %q, want static", rehearsal.Tier)
	}
	if out.Tier != domain.TierStatic {
		t.Fatalf("proposal tier = %q, want the tier that ran", out.Tier)
	}
	if out.Status != domain.StatusVerified {
		t.Fatalf("status = %q, want a partial rehearsal to stay reviewable", out.Status)
	}
}

// A rehearsal that proves nothing marks the proposal failed and names why — whether the engine
// explained itself or only left a note. The reviewer's next move is to fix the cause and re-run.
func TestRehearseMarksAProposalFailedWhenItProvesNothing(t *testing.T) {
	cases := []struct {
		name      string
		rehearsal domain.Rehearsal
		want      string
	}{
		{
			name: "the engine said why",
			rehearsal: domain.Rehearsal{
				Status: domain.RehearsalFailed, Tier: domain.TierShadow,
				Error: "the rollback left the schema different from where it started",
			},
			want: "rollback",
		},
		{
			name: "only a note",
			rehearsal: domain.Rehearsal{
				Status: domain.RehearsalUnsupported, Tier: domain.TierShadow,
				Note: "the statement could not be scoped to this target",
			},
			want: "unsupported",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t, domain.StatusDraft)
			f.driver.rehearsal = test.rehearsal

			out, _, err := NewRehearse(f.deps).Run(context.Background(), RehearseInput{Ref: f.proposal.Ref})
			if err != nil {
				t.Fatalf("Rehearse: %v", err)
			}
			if out.Status != domain.StatusFailed {
				t.Fatalf("status = %q, want failed", out.Status)
			}
			if !strings.Contains(out.Failure, test.want) {
				t.Fatalf("failure = %q, want it to mention %q", out.Failure, test.want)
			}
		})
	}
}

// A reference that is not in the state file is a not-found, not the store's foreign key reported as a
// server error: a caller can correct a wrong reference and cannot correct an internal one.
func TestCreateRefusesATargetThatIsNotThere(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	f.targets.err = domain.ErrNotFound
	seeded := len(f.proposals.stored)

	_, err := NewCreate(f.deps).Run(context.Background(), CreateInput{
		TargetRef: f.target.Ref, Title: "add a column",
		Query: "ALTER TABLE users ADD COLUMN note TEXT", Actor: "agent",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Create for a missing target: %v, want ErrNotFound", err)
	}
	if len(f.proposals.stored) != seeded {
		t.Fatalf("stored %d proposals, want the %d the fixture started with", len(f.proposals.stored), seeded)
	}
}

// A schema change undone by a data change cannot restore what it changed, so the pair is refused before
// it is stored — naming both kinds — instead of being filed and reported as a failed rollback later.
func TestCreateRefusesARollbackOfAnotherKind(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)
	seeded := len(f.proposals.stored)

	_, err := NewCreate(f.deps).Run(context.Background(), CreateInput{
		TargetRef: f.target.Ref, Title: "add a column",
		Query:    "ALTER TABLE users ADD COLUMN note TEXT",
		Rollback: "UPDATE users SET note = NULL",
		Actor:    "agent",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("Create with a mismatched rollback: %v, want ErrInvalid", err)
	}
	if !strings.Contains(err.Error(), "ddl") || !strings.Contains(err.Error(), "dml") {
		t.Fatalf("error = %q, want it to name both kinds", err)
	}
	if len(f.proposals.stored) != seeded {
		t.Fatalf("stored %d proposals, want the %d the fixture started with", len(f.proposals.stored), seeded)
	}
}

// The same rule the other way round: a data change is undone by a data change, and that pair is stored.
func TestCreateAcceptsADataChangeUndoneByData(t *testing.T) {
	f := newFixture(t, domain.StatusDraft)

	out, err := NewCreate(f.deps).Run(context.Background(), CreateInput{
		TargetRef: f.target.Ref, Title: "note every user",
		Query:    "UPDATE users SET note = 'seen'",
		Rollback: "UPDATE users SET note = NULL",
		Actor:    "agent",
	})
	if err != nil {
		t.Fatalf("Create with a matching rollback: %v", err)
	}
	if out.Proposal.Kind != domain.KindDML {
		t.Fatalf("kind = %q, want dml", out.Proposal.Kind)
	}
}

// journalEntry finds one line of the history by its kind, or nil: an apply writes two kinds of line now.
func journalEntry(entries []domain.JournalEntry, kind string) *domain.JournalEntry {
	for i := range entries {
		if entries[i].Kind == kind {
			return &entries[i]
		}
	}
	return nil
}

// A probe that fails after a successful apply is stored as a failed probe and never as a failed apply:
// the statement is committed, and what the connection thinks of the server is a different fact.
func TestApplyRecordsAFailedProbeWithoutFailingTheApply(t *testing.T) {
	f := newFixture(t, domain.StatusApproved)
	f.driver.receipt = domain.Receipt{Algorithm: domain.AlgorithmInstant, RowsAffected: 1}
	f.driver.probeErr = errors.New("dial tcp: connection refused")

	out, err := NewApply(f.deps).Run(context.Background(), ApplyInput{Ref: f.proposal.Ref, Actor: "ops"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Proposal.Status != domain.StatusApplied {
		t.Fatalf("status = %q, want applied", out.Proposal.Status)
	}
	probed := journalEntry(f.journal.entries, "probed")
	if probed == nil || probed.Detail["status"] != "failed" {
		t.Fatalf("journal = %+v, want a failed probe line", f.journal.entries)
	}
}

// Two applies against the same target are a queue, not a race: while one holds the target's lock, the
// other waits for it instead of fighting the engine's own DDL lock and losing.
func TestApplyWaitsForTheTarget(t *testing.T) {
	f := newFixture(t, domain.StatusApproved)
	f.deps.Locks = &keyed.Group{}
	unlock := f.deps.Locks.Lock(f.target.Ref)

	done := make(chan error, 1)
	go func() {
		_, err := NewApply(f.deps).Run(context.Background(), ApplyInput{Ref: f.proposal.Ref, Actor: "ops"})
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("Apply ran while the target was locked (err = %v), want it to wait", err)
	case <-time.After(50 * time.Millisecond):
	}

	unlock()
	if err := <-done; err != nil {
		t.Fatalf("Apply after the lock was released: %v", err)
	}
}
