package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"noxis/internal/domain"
)

// storedProposal writes one proposal in the status the test needs, against a target that exists.
func storedProposal(t *testing.T, state *DB, status domain.Status) domain.Proposal {
	t.Helper()
	ctx := context.Background()

	target, err := state.Create(ctx, newTarget(t, "prod"))
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	proposal := newProposal(t, target.Ref, "add a column")
	proposal.Status = status
	stored, err := state.CreateProposal(ctx, proposal)
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	return stored
}

// The claim is the lock. Two operators pressing apply at the same moment must not both run the
// statement, and the loser has to learn why.
func TestClaimIsTheLock(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	proposal := storedProposal(t, state, domain.StatusApproved)

	first, err := state.ClaimProposal(ctx, proposal.Ref, domain.StatusApproved, domain.StatusApplying)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if first.Status != domain.StatusApplying {
		t.Fatalf("first claim left it %s, want applying", first.Status)
	}

	if _, err := state.ClaimProposal(ctx, proposal.Ref, domain.StatusApproved, domain.StatusApplying); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second claim: %v, want ErrConflict", err)
	}

	// The loser's refusal must not have moved the row: it is still the winner's claim.
	current, err := state.ProposalByRef(ctx, proposal.Ref)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if current.Status != domain.StatusApplying {
		t.Fatalf("status = %s after a losing claim, want applying", current.Status)
	}
}

// Progress arrives while the receipt is being written, so it writes its own two columns and nothing
// else: a progress update that carried the whole row would overwrite the claim.
func TestSetProgressTouchesOnlyProgress(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	proposal := storedProposal(t, state, domain.StatusApproved)

	if _, err := state.ClaimProposal(ctx, proposal.Ref, domain.StatusApproved, domain.StatusApplying); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := state.SetProgress(ctx, proposal.Ref, domain.Progress{Stage: "copy to tmp table", Percent: 42}); err != nil {
		t.Fatalf("set progress: %v", err)
	}

	current, err := state.ProposalByRef(ctx, proposal.Ref)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if current.Progress.Stage != "copy to tmp table" || current.Progress.Percent != 42 {
		t.Fatalf("progress = %+v, want the reported stage and percent", current.Progress)
	}
	if current.Status != domain.StatusApplying {
		t.Fatalf("status = %s, want the claim to survive a progress update", current.Status)
	}
	if current.Query != proposal.Query || current.Title != proposal.Title {
		t.Fatal("a progress update rewrote the proposal")
	}
}

// A proposal is stored with its receipt fields round-tripping, including a nil started_at: the
// difference between "not started" and "started at the epoch" is a difference someone reads.
func TestProposalReceiptRoundTrip(t *testing.T) {
	state := open(t)
	ctx := context.Background()
	proposal := storedProposal(t, state, domain.StatusApproved)

	started := time.Now().UTC().Truncate(time.Millisecond)
	proposal.StartedAt = &started
	proposal.Algorithm = domain.AlgorithmCopy
	proposal.RowsEstimated = 2_000_000
	proposal.RowsAffected = 1_999_998
	proposal.Progress = domain.Progress{Stage: "done", Percent: 100}
	if _, err := state.UpdateProposal(ctx, proposal); err != nil {
		t.Fatalf("update: %v", err)
	}

	read, err := state.ProposalByRef(ctx, proposal.Ref)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Algorithm != domain.AlgorithmCopy || read.RowsEstimated != 2_000_000 || read.RowsAffected != 1_999_998 {
		t.Fatalf("receipt = %+v", read)
	}
	if read.StartedAt == nil || !read.StartedAt.Equal(started) {
		t.Fatalf("started_at = %v, want %s", read.StartedAt, started)
	}
	if read.Progress.Stage != "done" || read.Progress.Percent != 100 {
		t.Fatalf("progress = %+v", read.Progress)
	}
}
