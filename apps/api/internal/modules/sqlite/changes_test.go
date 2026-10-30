package sqlite

import (
	"bytes"
	"context"
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
	foreign := entity.Connection{Name: "elsewhere", Driver: constants.DriverPostgres}
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
