package changeflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

type testStore struct {
	changes []entity.Change
	saved   []entity.Change
	saveErr error
	listErr error
}

func (s *testStore) ListChanges(context.Context) ([]entity.Change, error) {
	return append([]entity.Change(nil), s.changes...), s.listErr
}
func (*testStore) GetChange(context.Context, string) (*entity.Change, error) { return nil, nil }
func (s *testStore) SaveChange(_ context.Context, change entity.Change) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = append(s.saved, change)
	return nil
}
func (*testStore) ClaimChange(context.Context, string, int64, string) (bool, error) { return true, nil }
func (*testStore) DeleteChange(context.Context, string) error                       { return nil }

func TestApproveAndRejectRules(t *testing.T) {
	for _, status := range []string{constants.ChangeValidated, constants.ChangeUnverified} {
		store := &testStore{}
		change := entity.Change{ID: "change", Status: status}
		if err := New(store, constants.FamilySQL).Approve(context.Background(), &change, "reviewer"); err != nil {
			t.Fatalf("approve %s: %v", status, err)
		}
		if change.Status != constants.ChangeApproved || change.ApprovedAt == 0 || change.ApprovedBy != "reviewer" || len(store.saved) != 1 {
			t.Fatalf("approved change = %+v, saves=%d", change, len(store.saved))
		}
	}

	store := &testStore{}
	flow := New(store, constants.FamilySQL)
	applying := entity.Change{Status: constants.ChangeApplying}
	if err := flow.Reject(context.Background(), &applying, "reviewer"); !errors.Is(err, constants.ErrConflict) {
		t.Fatalf("reject applying error = %v, want conflict", err)
	}
	applied := entity.Change{Status: constants.ChangeApplied}
	if err := flow.Reject(context.Background(), &applied, "reviewer"); err == nil {
		t.Fatal("reject applied change succeeded")
	}
	rejected := entity.Change{Status: constants.ChangeValidated}
	if err := flow.Reject(context.Background(), &rejected, "reviewer"); err != nil || rejected.RejectedBy != "reviewer" {
		t.Fatalf("reject actor=%q error=%v", rejected.RejectedBy, err)
	}
}

func TestScheduleApplyableAndDueBoundaries(t *testing.T) {
	flow := New(&testStore{}, constants.FamilySQL)
	change := entity.Change{ID: "change", Status: constants.ChangeApproved}
	future := time.Now().Add(time.Minute).Unix()
	if err := flow.Schedule(context.Background(), &change, future, "reviewer"); err != nil {
		t.Fatalf("schedule future: %v", err)
	}
	if change.ScheduledBy != "reviewer" {
		t.Fatalf("scheduled by=%q, want reviewer", change.ScheduledBy)
	}
	if err := Applyable(&change, future-1); err == nil {
		t.Fatal("future schedule was applyable")
	}
	if err := Applyable(&change, future); err != nil {
		t.Fatalf("due schedule was not applyable: %v", err)
	}
	if !Due(change, future) || Due(change, future-1) {
		t.Fatalf("due boundary at %d is wrong", future)
	}
	if err := flow.Schedule(context.Background(), &change, 0, "reviewer"); err != nil {
		t.Fatalf("clear schedule: %v", err)
	}
	if Due(change, future) {
		t.Fatal("immediate change was due")
	}
	change.Status = constants.ChangeApplying
	if err := Applyable(&change, future); !errors.Is(err, constants.ErrConflict) {
		t.Fatalf("applying state error = %v, want conflict", err)
	}
}

func TestRecordAndTick(t *testing.T) {
	store := &testStore{changes: []entity.Change{
		{ID: "failed", Kind: constants.KindDML, Status: constants.ChangeApproved, ScheduledAt: 99},
		{ID: "applied", Kind: constants.KindDML, Status: constants.ChangeApproved, ScheduledAt: 100},
		{ID: "claimed", Kind: constants.KindDML, Status: constants.ChangeApproved, ScheduledAt: 100},
		{ID: "other-kind", Kind: constants.KindMongo, Status: constants.ChangeApproved, ScheduledAt: 100},
		{ID: "future", Kind: constants.KindDML, Status: constants.ChangeApproved, ScheduledAt: 101},
	}}
	flow := New(store, constants.FamilySQL)
	failed, err := flow.Record(context.Background(), entity.Change{ID: "rejected", Status: constants.ChangeFailed})
	if err != nil || failed.ID != "" || len(store.saved) != 0 {
		t.Fatalf("failed submission record = %+v, %v; saves=%d", failed, err, len(store.saved))
	}
	if _, err := flow.Record(context.Background(), entity.Change{ID: "valid", Status: constants.ChangeValidated}); err != nil || len(store.saved) != 1 {
		t.Fatalf("valid submission record error=%v saves=%d", err, len(store.saved))
	}

	result, err := Tick(context.Background(), store, 100, func(change entity.Change) bool {
		return change.Kind == constants.KindDML
	}, func(_ context.Context, change *entity.Change) (entity.Change, error) {
		switch change.ID {
		case "failed":
			return *change, errors.New("engine failure")
		case "claimed":
			return *change, constants.ErrConflict
		default:
			return *change, nil
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != 1 || result.Applied[0] != "applied" || len(result.Failed) != 1 || result.Failed[0] != "failed" {
		t.Fatalf("tick result = %+v", result)
	}
}
