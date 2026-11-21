package sqlite

import (
	"context"
	"errors"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"testing"
)

// The unique index is what settles a race, and ByKey is what a retry reads. Two creates under one key
// answer one row — the first one — and a key nobody used is a not-found rather than an empty row.
func TestIdempotencyKeyAnswersTheFirstRow(t *testing.T) {
	state := open(t)
	ctx := context.Background()

	first := newTarget(t, "prod")
	first.IdempotencyKey = "retry-me"
	created, err := state.Create(ctx, first)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// A retry is a new registration racing the first one: a different name, the same key.
	retry := newTarget(t, "prod-again")
	retry.IdempotencyKey = "retry-me"
	replayed, err := state.Create(ctx, retry)
	if err != nil {
		t.Fatalf("create under a used key failed instead of answering the first row: %v", err)
	}
	if replayed.Ref != created.Ref {
		t.Fatalf("refs = %q and %q, want the row the first call made", created.Ref, replayed.Ref)
	}

	found, err := state.ByKey(ctx, "retry-me")
	if err != nil {
		t.Fatalf("ByKey: %v", err)
	}
	if found.Ref != created.Ref {
		t.Fatalf("ByKey answered %q, want %q", found.Ref, created.Ref)
	}
	if _, err := state.ByKey(ctx, "never-used"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ByKey for an unused key = %v, want ErrNotFound", err)
	}

	// The same rule on the other create: a change proposed twice under one key is one change.
	proposal := newProposal(t, created.Ref, "add a column")
	proposal.IdempotencyKey = "change-retry"
	stored, err := state.CreateProposal(ctx, proposal)
	if err != nil {
		t.Fatalf("first propose: %v", err)
	}
	again := newProposal(t, created.Ref, "add another column")
	again.IdempotencyKey = "change-retry"
	replayedProposal, err := state.CreateProposal(ctx, again)
	if err != nil {
		t.Fatalf("propose under a used key: %v", err)
	}
	if replayedProposal.Ref != stored.Ref {
		t.Fatalf("proposal refs = %q and %q, want the first one", stored.Ref, replayedProposal.Ref)
	}
	if byKey, err := state.ProposalByKey(ctx, "change-retry"); err != nil || byKey.Ref != stored.Ref {
		t.Fatalf("ProposalByKey = %q, %v", byKey.Ref, err)
	}
}

// Two targets created without a key are two targets: the index tolerates the many rows that named no
// request, which is only true because the empty key is stored as NULL.
func TestTargetsWithoutAKeyAreAllDistinct(t *testing.T) {
	state := open(t)
	ctx := context.Background()

	for _, name := range []string{"prod", "staging"} {
		if _, err := state.Create(ctx, newTarget(t, name)); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	targets, err := state.List(ctx, ports.TargetFilter{Limit: 10, IncludeArchived: true})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("listed %d targets, want 2", len(targets))
	}
}
