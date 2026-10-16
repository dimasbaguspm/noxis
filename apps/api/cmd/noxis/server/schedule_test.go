package server

import (
	"context"
	"testing"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// Only an approved change is schedulable, a past slot is refused, and 0 clears
// the schedule.
func TestScheduleApprovedChange(t *testing.T) {
	d, store, _ := newTestDeps(&fakeSQL{})
	ctx := context.Background()
	store.changes["x"] = entity.Change{ID: "x", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeValidated}

	if _, err := d.Schedule.Run(ctx, "x", time.Now().Add(time.Hour).Unix()); err == nil {
		t.Fatal("an unapproved change must not be schedulable")
	}

	store.changes["x"] = entity.Change{ID: "x", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeApproved}
	at := time.Now().Add(time.Hour).Unix()
	got, err := d.Schedule.Run(ctx, "x", at)
	if err != nil || got.ScheduledAt != at {
		t.Fatalf("schedule = %+v, %v", got, err)
	}

	if _, err := d.Schedule.Run(ctx, "x", time.Now().Add(-time.Minute).Unix()); err == nil {
		t.Fatal("a past slot must be refused")
	}

	cleared, err := d.Schedule.Run(ctx, "x", 0)
	if err != nil || cleared.ScheduledAt != 0 {
		t.Fatalf("clear = %+v, %v", cleared, err)
	}
}
