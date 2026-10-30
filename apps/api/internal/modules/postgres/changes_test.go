package postgres

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/mocks"
)

func TestReviewDecisionsLogActor(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	controller := gomock.NewController(t)
	store := mocks.NewMockChangeRepository(controller)
	store.EXPECT().SaveChange(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	flow := NewChanges(nil, store, nil, nil)
	ctx := context.Background()

	approved := entity.Change{ID: "approve-1", Connection: "prod", Status: constants.ChangeValidated}
	if err := flow.Approve(ctx, &approved, "local:reviewer"); err != nil {
		t.Fatal(err)
	}
	rejected := entity.Change{ID: "reject-1", Connection: "prod", Status: constants.ChangeValidated}
	if err := flow.Reject(ctx, &rejected, "local:reviewer"); err != nil {
		t.Fatal(err)
	}
	scheduled := entity.Change{ID: "schedule-1", Connection: "prod", Status: constants.ChangeApproved}
	if err := flow.Schedule(ctx, &scheduled, time.Now().Add(time.Minute).Unix(), "local:reviewer"); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"change approved", "change rejected", "change schedule updated", "local:reviewer", "approve-1", "reject-1", "schedule-1"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("decision logs do not contain %q: %s", want, output.String())
		}
	}
}

// TestTickLeavesAnotherDriversChangeAlone proves each driver's queue ticks only its own connections: with one queue per driver, a queue that claimed another driver's change would apply it through the wrong engine.
func TestTickLeavesAnotherDriversChangeAlone(t *testing.T) {
	controller := gomock.NewController(t)
	store := mocks.NewMockChangeRepository(controller)
	conns := mocks.NewMockConnectionRepository(controller)
	foreign := entity.Connection{Name: "elsewhere", Driver: constants.DriverMySQL}
	conns.EXPECT().Get(gomock.Any(), "elsewhere").Return(&foreign, nil).AnyTimes()
	store.EXPECT().ListChanges(gomock.Any()).Return([]entity.Change{{
		ID: "scheduled-1", Connection: "elsewhere", Kind: constants.KindDML,
		Status: constants.ChangeApproved, ScheduledAt: 1,
	}}, nil)

	res, err := NewChanges(conns, store, nil, nil).Tick(context.Background(), 2, "local:ticker")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 0 || len(res.Failed) != 0 {
		t.Fatalf("another driver's change was claimed: %+v", res)
	}
}

// TestTickRunsDueChangesOnce locks the rest of the loop: a change runs when its slot arrives, a future one waits, a lost claim is skipped rather than failed, and an engine refusal is reported.
func TestTickRunsDueChangesOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockChangeRepository(ctrl)
	conns := mocks.NewMockConnectionRepository(ctrl)
	engine := mocks.NewMockSQLChangeEngine(ctrl)

	store.EXPECT().ListChanges(gomock.Any()).Return([]entity.Change{
		{ID: "due", Connection: "demo", Query: "CREATE TABLE due (id INT)", Kind: constants.KindDDL, Status: constants.ChangeApproved, ScheduledAt: 100},
		{ID: "future", Connection: "demo", Kind: constants.KindDDL, Status: constants.ChangeApproved, ScheduledAt: 101},
		{ID: "claimed", Connection: "demo", Kind: constants.KindDDL, Status: constants.ChangeApproved, ScheduledAt: 100},
		{ID: "failing", Connection: "demo", Query: "CREATE TABLE nope (id INT)", Kind: constants.KindDDL, Status: constants.ChangeApproved, ScheduledAt: 100},
	}, nil)
	conns.EXPECT().Get(gomock.Any(), "demo").Return(&entity.Connection{Name: "demo", Driver: constants.DriverPostgres}, nil).AnyTimes()
	store.EXPECT().ClaimChange(gomock.Any(), "due", gomock.Any(), constants.SchedulerActor).Return(true, nil)
	store.EXPECT().ClaimChange(gomock.Any(), "claimed", gomock.Any(), constants.SchedulerActor).Return(false, nil)
	store.EXPECT().ClaimChange(gomock.Any(), "failing", gomock.Any(), constants.SchedulerActor).Return(true, nil)
	store.EXPECT().SaveChange(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	engine.EXPECT().Apply(gomock.Any(), gomock.Any(), "CREATE TABLE due (id INT)").Return(entity.Result{Affected: 1}, nil)
	engine.EXPECT().Apply(gomock.Any(), gomock.Any(), "CREATE TABLE nope (id INT)").Return(entity.Result{}, errors.New("engine said no"))

	result, err := NewChanges(conns, store, engine, nil).Tick(context.Background(), 100, constants.SchedulerActor)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(result.Applied) != 1 || result.Applied[0] != "due" {
		t.Fatalf("applied = %v, want only the due change", result.Applied)
	}
	if len(result.Failed) != 1 || result.Failed[0] != "failing" {
		t.Fatalf("failed = %v, want only the change the engine refused", result.Failed)
	}
}
