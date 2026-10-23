package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

func TestClaimChangeIsAtomicAndProtectsApplyingState(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	change := entity.Change{
		ID: "change-1", Connection: "demo", Notes: "change", Query: "INSERT", Rollback: "DELETE",
		Kind: constants.KindDML, Status: constants.ChangeApproved, CreatedAt: 1,
	}
	if err := store.SaveChange(ctx, change); err != nil {
		t.Fatalf("save: %v", err)
	}

	type claimResult struct {
		at      int64
		claimed bool
		err     error
	}
	start := make(chan struct{})
	results := make(chan claimResult, 2)
	for _, at := range []int64{20, 21} {
		go func(at int64) {
			<-start
			claimed, err := store.ClaimChange(ctx, change.ID, at)
			results <- claimResult{at: at, claimed: claimed, err: err}
		}(at)
	}
	close(start)
	first, second := <-results, <-results
	winner := first
	if !winner.claimed {
		winner = second
	}
	if first.err != nil || second.err != nil || first.claimed == second.claimed {
		t.Fatalf("concurrent claims = %+v and %+v; want one winner and no errors", first, second)
	}

	change.Status = constants.ChangeRejected
	if err := store.SaveChange(ctx, change); !errors.Is(err, constants.ErrConflict) {
		t.Fatalf("stale save error = %v, want ErrConflict", err)
	}
	stored, err := store.GetChange(ctx, change.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.Status != constants.ChangeApplying || stored.ApplyingAt != winner.at {
		t.Fatalf("stored state = %q at %d, want applying at winning claim time %d", stored.Status, stored.ApplyingAt, winner.at)
	}
}
