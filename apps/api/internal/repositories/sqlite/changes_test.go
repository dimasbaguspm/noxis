package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

func TestOpenAddsChangeActorColumnsToExistingStore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "noxis.db")
	store, err := Open(ctx, path, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"approved_by", "applied_by", "rejected_by", "scheduled_by"} {
		if _, err := store.db.Exec(`ALTER TABLE changes DROP COLUMN ` + column); err != nil {
			t.Fatalf("drop %s to simulate an older store: %v", column, err)
		}
	}
	store.Close()

	store, err = Open(ctx, path, "")
	if err != nil {
		t.Fatalf("reopen existing store: %v", err)
	}
	defer store.Close()
	rows, err := store.db.Query(`PRAGMA table_info(changes)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"approved_by", "applied_by", "rejected_by", "scheduled_by"} {
		if !columns[column] {
			t.Errorf("migration did not restore %s", column)
		}
	}
}

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
		CreatedBy: "creator", ApprovedBy: "alice", ScheduledAt: 1_000_000_000, ScheduledBy: "scheduler",
	}
	if err := store.SaveChange(ctx, change); err != nil {
		t.Fatalf("save: %v", err)
	}

	type claimResult struct {
		at      int64
		actor   string
		claimed bool
		err     error
	}
	start := make(chan struct{})
	results := make(chan claimResult, 2)
	for _, attempt := range []claimResult{{at: 20, actor: "alice"}, {at: 21, actor: "bob"}} {
		go func(attempt claimResult) {
			<-start
			claimed, err := store.ClaimChange(ctx, change.ID, attempt.at, attempt.actor)
			attempt.claimed, attempt.err = claimed, err
			results <- attempt
		}(attempt)
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
	if stored.Status != constants.ChangeApplying || stored.ApplyingAt != winner.at || stored.AppliedBy != winner.actor {
		t.Fatalf("stored state = %q at %d by %q, want applying at winning claim time %d by %q", stored.Status, stored.ApplyingAt, stored.AppliedBy, winner.at, winner.actor)
	}
	if stored.CreatedBy != "creator" || stored.ApprovedBy != "alice" || stored.ScheduledBy != "scheduler" {
		t.Fatalf("persisted actors = created:%q approved:%q scheduled:%q", stored.CreatedBy, stored.ApprovedBy, stored.ScheduledBy)
	}
}
