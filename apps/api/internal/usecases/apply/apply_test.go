package apply

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"noxis/internal/common/testkit"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/mocks"
	"noxis/internal/modules"
	"noxis/internal/modules/sql"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

func approved(id string) entity.Change {
	return entity.Change{ID: id, Connection: "demo", Kind: constants.KindDDL, Status: constants.ChangeApproved, Query: "CREATE TABLE things (id INT)"}
}

// harness holds the change store and the SQL module, both mocked.
type harness struct {
	deps        usecases.RegistryDeps
	sq          *mocks.MockSQLModule
	changes     *mocks.MockChangeRepository
	stored      map[string]entity.Change
	saveErr     error
	claimDenied bool
}

func newHarness(t *testing.T, store ...entity.Change) *harness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &harness{
		sq:      mocks.NewMockSQLModule(ctrl),
		changes: mocks.NewMockChangeRepository(ctrl),
		stored:  map[string]entity.Change{},
	}
	for i := range store {
		h.stored[store[i].ID] = store[i]
	}
	conns := mocks.NewMockConnectionRepository(ctrl)
	conns.EXPECT().Get(gomock.Any(), "demo").Return(&entity.Connection{
		Name: "demo", Driver: constants.DriverPostgres,
	}, nil).AnyTimes()
	h.changes.EXPECT().GetChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id string) (*entity.Change, error) {
		c, ok := h.stored[id]
		if !ok {
			return nil, constants.ErrNotFound
		}
		return &c, nil
	}).AnyTimes()
	h.changes.EXPECT().ClaimChange(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id string, at int64) (bool, error) {
		if h.claimDenied {
			return false, nil
		}
		c, ok := h.stored[id]
		if !ok || c.Status != constants.ChangeApproved {
			return false, nil
		}
		c.Status, c.ApplyingAt = constants.ChangeApplying, at
		h.stored[id] = c
		return true, nil
	}).AnyTimes()
	h.changes.EXPECT().SaveChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c entity.Change) error {
		if h.saveErr != nil {
			return h.saveErr
		}
		h.stored[c.ID] = c
		return nil
	}).AnyTimes()
	h.changes.EXPECT().ListChanges(gomock.Any()).DoAndReturn(func(context.Context) ([]entity.Change, error) {
		out := make([]entity.Change, 0, len(h.stored))
		for id := range h.stored {
			out = append(out, h.stored[id])
		}
		return out, nil
	}).AnyTimes()
	h.sq.EXPECT().Family().Return(constants.FamilySQL).AnyTimes()
	h.sq.EXPECT().Supports(constants.DriverPostgres).Return(true).AnyTimes()
	registry := modules.New(
		map[string]interfaces.Module{constants.FamilySQL: h.sq},
		interfaces.ChangeFlows{SQL: sql.NewChanges(conns, h.changes, h.sq, nil)},
		nil,
	)
	h.deps = usecases.RegistryDeps{
		Registry: repositories.Registry{Connections: conns, Changes: h.changes},
		Modules:  registry,
	}
	return h
}

func TestApplyAfterApprove(t *testing.T) {
	h := newHarness(t, approved("x"))
	h.sq.EXPECT().Apply(gomock.Any(), gomock.Any(), gomock.Any()).Return(entity.Result{Affected: 1, Message: "applied: 1 statement(s)"}, nil)

	mux := testkit.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/apply", Apply{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/engine/sql/changes/x/apply", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := h.stored["x"].Status; got != constants.ChangeApplied {
		t.Fatalf("status=%q want applied", got)
	}
}

func TestApplyFailureIsRecorded(t *testing.T) {
	h := newHarness(t, approved("x"))
	h.sq.EXPECT().Apply(gomock.Any(), gomock.Any(), gomock.Any()).Return(entity.Result{}, errors.New("statement 1: duplicate key"))

	mux := testkit.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/apply", Apply{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/engine/sql/changes/x/apply", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code == http.StatusOK {
		t.Fatalf("apply must surface the failure: %s", rec.Body.String())
	}
	stored := h.stored["x"]
	if stored.Status != constants.ChangeFailed {
		t.Fatalf("failure must be recorded: %+v", stored)
	}
	last := stored.Checks[len(stored.Checks)-1]
	if last.OK || !strings.Contains(last.Error, "duplicate key") {
		t.Fatalf("apply check: %+v", last)
	}
}

func TestApplyFailureReportsOutcomeSaveFailure(t *testing.T) {
	h := newHarness(t, approved("x"))
	h.saveErr = errors.New("sqlite unavailable")
	h.sq.EXPECT().Apply(gomock.Any(), gomock.Any(), gomock.Any()).Return(entity.Result{}, errors.New("duplicate key"))

	mux := testkit.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/apply", Apply{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/engine/sql/changes/x/apply", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), "sqlite unavailable") || !strings.Contains(rec.Body.String(), "duplicate key") {
		t.Fatalf("apply must surface both errors: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := h.stored["x"].Status; got != constants.ChangeApplying {
		t.Fatalf("stored status=%q want applying when failed outcome cannot be saved", got)
	}
}

func TestApplyConflictDoesNotRunEngine(t *testing.T) {
	h := newHarness(t, approved("x"))
	h.claimDenied = true
	mux := testkit.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/apply", Apply{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/engine/sql/changes/x/apply", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s, want 409", rec.Code, rec.Body.String())
	}
}

func TestTickSkipsChangeClaimedElsewhere(t *testing.T) {
	due := approved("due")
	due.ScheduledAt = time.Now().Add(-time.Minute).Unix()
	h := newHarness(t, due)
	h.claimDenied = true

	rec := testkit.Do(testkit.Mux("POST", "/internal/callback/scheduler", Tick{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/internal/callback/scheduler", "", nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"due"`) {
		t.Fatalf("tick status=%d body=%s; a concurrent claim is not an apply failure", rec.Code, rec.Body.String())
	}
}

func TestTickAppliesDueScheduledChange(t *testing.T) {
	due := approved("due")
	due.ScheduledAt = time.Now().Add(-time.Minute).Unix()
	h := newHarness(t, due, approved("wait")) // "wait" is immediate: not due
	h.sq.EXPECT().Apply(gomock.Any(), gomock.Any(), gomock.Any()).Return(entity.Result{Message: "applied"}, nil)

	rec := testkit.Do(testkit.Mux("POST", "/internal/callback/scheduler", Tick{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/internal/callback/scheduler", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"due"`) {
		t.Fatalf("tick status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := h.stored["due"].Status; got != constants.ChangeApplied {
		t.Fatalf("due status=%q want applied", got)
	}
	if got := h.stored["wait"].Status; got != constants.ChangeApproved {
		t.Fatalf("immediate status=%q want approved", got)
	}
}
