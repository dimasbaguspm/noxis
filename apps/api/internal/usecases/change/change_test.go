package change

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"noxis/internal/common"
	"noxis/internal/common/testkit"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/mocks"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

func admin() *entity.Identity { return &entity.Identity{Subject: "root", Role: entity.RoleSuper} }

const sampleSQL = `{"connection":"demo","notes":"add things","query":"CREATE TABLE things (id INT)","rollback":"DROP TABLE things"}`

// harness is the change use case's dependency set: one SQL module whose validation outcome the test controls, and a super admin's connection.
type harness struct {
	deps    usecases.RegistryDeps
	sq      *mocks.MockSQLChangeEngine
	conns   *mocks.MockConnectionRepository
	changes *mocks.MockChangeRepository
}

// newHarness wires the mocks and the one connection every change refers to.
func newHarness(t *testing.T) *harness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &harness{
		sq:      mocks.NewMockSQLChangeEngine(ctrl),
		conns:   mocks.NewMockConnectionRepository(ctrl),
		changes: mocks.NewMockChangeRepository(ctrl),
	}
	h.sq.EXPECT().Driver().Return(constants.DriverPostgres).AnyTimes()
	h.sq.EXPECT().Supports(constants.DriverPostgres).Return(true).AnyTimes()
	h.sq.EXPECT().Kind(gomock.Any()).Return(constants.KindDDL).AnyTimes()
	h.sq.EXPECT().Referenced(gomock.Any()).Return([]string{"things"}, []string{"id"}).AnyTimes()
	h.conns.EXPECT().Get(gomock.Any(), "demo").Return(&entity.Connection{
		Name: "demo", Driver: constants.DriverPostgres,
	}, nil).AnyTimes()
	registry := modules.New(map[string]interfaces.Module{constants.DriverPostgres: h.sq})
	h.deps = usecases.RegistryDeps{
		Repositories: repositories.Registry{Connections: h.conns, Changes: h.changes},
		Modules:      registry,
	}
	return h
}

func (h *harness) do(handler http.Handler, method, path, body string, p *entity.Identity) *httptest.ResponseRecorder {
	ctx := common.WithModuleLoader(context.Background(), h.deps.Modules)
	return testkit.DoContext(ctx, handler, method, path, body, p)
}

func TestFilterByConnection(t *testing.T) {
	changes := []entity.Change{{Connection: "prod-db"}, {Connection: "cache"}, {Connection: "prod-cache"}}
	cases := []struct {
		needle string
		want   int
	}{
		{"", 3},
		{"PROD", 2},
		{"cache", 2},
		{"nope", 0},
	}
	for _, tc := range cases {
		if got := filterByConnection(changes, tc.needle); len(got) != tc.want {
			t.Fatalf("filter %q = %d, want %d", tc.needle, len(got), tc.want)
		}
	}
}

// TestSubmitWithoutRegisteredEngineFailsClosed proves a submit never guesses an engine: with no module registered for
// the driver in the path, the request is refused instead of being recorded against nothing.
func TestSubmitWithoutRegisteredEngineFailsClosed(t *testing.T) {
	h := newHarness(t)
	deps := h.deps
	deps.Modules = modules.New(map[string]interfaces.Module{})
	rec := testkit.Do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", sampleSQL, admin())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unregistered engine status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSubmitThenApprove(t *testing.T) {
	h := newHarness(t)
	h.sq.EXPECT().VerifyWrite(gomock.Any()).Return(nil).AnyTimes()
	h.sq.EXPECT().ValidateChange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(entity.Result{Message: "query and rollback both valid (rolled back)"}, nil)
	saved := entity.Change{}
	h.changes.EXPECT().SaveChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c entity.Change) error {
		saved = c
		return nil
	}).Times(2)
	h.changes.EXPECT().GetChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string) (*entity.Change, error) {
		c := saved
		return &c, nil
	}).AnyTimes()

	rec := h.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", sampleSQL, admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", rec.Code, rec.Body.String())
	}
	var ch entity.Change
	if err := json.Unmarshal(rec.Body.Bytes(), &ch); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ch.Status != constants.ChangeValidated || ch.ID == "" || len(ch.Checks) != 2 {
		t.Fatalf("submit result: %+v", ch)
	}

	rec = h.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes/{id}/approve", Approve{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes/"+ch.ID+"/approve", "", admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", rec.Code, rec.Body.String())
	}
	var approved entity.Change
	_ = json.Unmarshal(rec.Body.Bytes(), &approved)
	if approved.Status != constants.ChangeApproved || approved.ApprovedAt == 0 || approved.ApprovedBy != "root" {
		t.Fatalf("approve result: %+v", approved)
	}
}

func TestSubmitValidationFailures(t *testing.T) {
	// a read on the change path is refused up front
	h := newHarness(t)
	h.sq.EXPECT().VerifyWrite(gomock.Any()).Return(errors.New(`statement "SELECT" not allowed here`)).AnyTimes()
	// the dry run still runs; the kind check is what already failed
	h.sq.EXPECT().ValidateChange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(entity.Result{}, nil)
	rec := h.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", sampleSQL, admin())
	var failed entity.Change
	_ = json.Unmarshal(rec.Body.Bytes(), &failed)
	if failed.Status != constants.ChangeFailed || failed.ID != "" {
		t.Fatalf("write failure: %+v", failed)
	}

	// a dry-run failure lands as failed with the reason recorded
	h2 := newHarness(t)
	h2.sq.EXPECT().VerifyWrite(gomock.Any()).Return(nil).AnyTimes()
	h2.sq.EXPECT().ValidateChange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(entity.Result{}, errors.New("rollback statement 1: syntax error"))
	rec = h2.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: h2.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", sampleSQL, admin())
	_ = json.Unmarshal(rec.Body.Bytes(), &failed)
	if failed.Status != constants.ChangeFailed || len(failed.Checks) < 2 || !strings.Contains(failed.Checks[1].Error, "syntax error") {
		t.Fatalf("dry-run failure: %+v", failed)
	}

	// mysql DDL cannot be dry-run: unverified, still approvable
	h3 := newHarness(t)
	h3.sq.EXPECT().VerifyWrite(gomock.Any()).Return(nil).AnyTimes()
	h3.sq.EXPECT().ValidateChange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(entity.Result{}, interfaces.ErrUnverifiable)
	unverified := entity.Change{}
	h3.changes.EXPECT().SaveChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c entity.Change) error {
		unverified = c
		return nil
	}).Times(2)
	h3.changes.EXPECT().GetChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string) (*entity.Change, error) {
		c := unverified
		return &c, nil
	}).AnyTimes()
	rec = h3.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: h3.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", sampleSQL, admin())
	var submitted entity.Change
	_ = json.Unmarshal(rec.Body.Bytes(), &submitted)
	if submitted.Status != constants.ChangeUnverified {
		t.Fatalf("unverified: %+v", submitted)
	}
	rec = h3.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes/{id}/approve", Approve{RegistryDeps: h3.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes/"+submitted.ID+"/approve", "", admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("unverified must be approvable: %d %s", rec.Code, rec.Body.String())
	}
}

func TestKindGated(t *testing.T) {
	h := newHarness(t)
	h.changes.EXPECT().GetChange(gomock.Any(), "m1").Return(&entity.Change{
		ID: "m1", Kind: constants.KindMongo, Status: constants.ChangeValidated, Connection: "demo",
	}, nil)
	rec := h.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes/{id}/approve", Approve{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes/m1/approve", "", admin())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sql engine must not approve a mongo change: %d %s", rec.Code, rec.Body.String())
	}
}

// TestListScopesToTheDriverInThePath proves the queue answers for the driver the route names, not for every SQL driver.
func TestListScopesToTheDriverInThePath(t *testing.T) {
	h := newHarness(t)
	h.conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{
		{Name: "demo", Driver: constants.DriverPostgres},
		{Name: "other", Driver: constants.DriverMySQL},
	}, nil).AnyTimes()
	h.changes.EXPECT().ListChanges(gomock.Any()).Return([]entity.Change{
		{ID: "p1", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeValidated},
		{ID: "m1", Connection: "other", Kind: constants.KindDML, Status: constants.ChangeValidated},
	}, nil)

	rec := h.do(testkit.Mux("GET", "/external/v1/engine/{driver}/changes", List{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/external/v1/engine/postgres/changes", "", admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got entity.ChangeList
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Total != 1 || len(got.Changes) != 1 || got.Changes[0].ID != "p1" {
		t.Fatalf("queue must hold only the postgres change: %+v", got)
	}
	if got.Changes[0].Driver != constants.DriverPostgres {
		t.Fatalf("change must name the driver that serves it: %+v", got.Changes[0])
	}
}

// TestGetRefusesAnotherDriversChange keeps a change reachable only through the driver of its own connection.
func TestGetRefusesAnotherDriversChange(t *testing.T) {
	h := newHarness(t)
	h.conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{
		{Name: "demo", Driver: constants.DriverMySQL},
	}, nil).AnyTimes()
	h.changes.EXPECT().GetChange(gomock.Any(), "m1").Return(&entity.Change{
		ID: "m1", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeValidated,
	}, nil)

	rec := h.do(testkit.Mux("GET", "/external/v1/engine/{driver}/changes/{id}", Get{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/external/v1/engine/postgres/changes/m1", "", admin())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("another driver's change must be not found: %d %s", rec.Code, rec.Body.String())
	}
}

// TestListFiltersByKind keeps the two queues apart: the filter runs before pagination, so the total is the queue's own count.
func TestListFiltersByKind(t *testing.T) {
	h := newHarness(t)
	h.conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{
		{Name: "demo", Driver: constants.DriverPostgres},
	}, nil).AnyTimes()
	h.changes.EXPECT().ListChanges(gomock.Any()).Return([]entity.Change{
		{ID: "d1", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeValidated},
		{ID: "x1", Connection: "demo", Kind: constants.KindDDL, Status: constants.ChangeValidated},
	}, nil).AnyTimes()

	for _, want := range []string{constants.KindDML, constants.KindDDL} {
		rec := h.do(testkit.Mux("GET", "/external/v1/engine/{driver}/changes", List{RegistryDeps: h.deps}.Serve),
			http.MethodGet, "/external/v1/engine/postgres/changes?kind="+want, "", admin())
		if rec.Code != http.StatusOK {
			t.Fatalf("kind=%s status=%d body=%s", want, rec.Code, rec.Body.String())
		}
		var got entity.ChangeList
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Total != 1 || len(got.Changes) != 1 || got.Changes[0].Kind != want {
			t.Fatalf("kind=%s must hold only its own change: %+v", want, got)
		}
	}
}

// TestListRefusesAnUnknownKind keeps a typo from silently returning the whole queue.
func TestListRefusesAnUnknownKind(t *testing.T) {
	h := newHarness(t)
	h.conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{
		{Name: "demo", Driver: constants.DriverPostgres},
	}, nil).AnyTimes()
	h.changes.EXPECT().ListChanges(gomock.Any()).Return([]entity.Change{}, nil).AnyTimes()

	rec := h.do(testkit.Mux("GET", "/external/v1/engine/{driver}/changes", List{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/external/v1/engine/postgres/changes?kind=mongo", "", admin())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown kind must be a bad request: %d %s", rec.Code, rec.Body.String())
	}
}

// TestListSearchesNotesAndStatements keeps the queue's name field honest: it matches a note or the statement itself.
func TestListSearchesNotesAndStatements(t *testing.T) {
	h := newHarness(t)
	h.conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{
		{Name: "demo", Driver: constants.DriverPostgres},
	}, nil).AnyTimes()
	h.changes.EXPECT().ListChanges(gomock.Any()).Return([]entity.Change{
		{ID: "a", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeValidated, Notes: "backfill emails"},
		{ID: "b", Connection: "demo", Kind: constants.KindDML, Status: constants.ChangeValidated, Notes: "other", Query: "UPDATE users SET x = 1"},
	}, nil).AnyTimes()

	for _, tc := range []struct{ q, want string }{
		{"emails", "a"},
		{"update users", "b"},
	} {
		rec := h.do(testkit.Mux("GET", "/external/v1/engine/{driver}/changes", List{RegistryDeps: h.deps}.Serve),
			http.MethodGet, "/external/v1/engine/postgres/changes?q="+url.QueryEscape(tc.q), "", admin())
		if rec.Code != http.StatusOK {
			t.Fatalf("q=%s status=%d body=%s", tc.q, rec.Code, rec.Body.String())
		}
		var got entity.ChangeList
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Total != 1 || len(got.Changes) != 1 || got.Changes[0].ID != tc.want {
			t.Fatalf("q=%s must match one change: %+v", tc.q, got)
		}
	}
}

// TestSubmitRefusesAMismatchedKind keeps a page from re-filing a change: the declared kind must match the statement's.
func TestSubmitRefusesAMismatchedKind(t *testing.T) {
	h := newHarness(t)
	h.changes.EXPECT().SaveChange(gomock.Any(), gomock.Any()).Times(0)

	const body = `{"connection":"demo","notes":"x","query":"CREATE TABLE things (id INT)","rollback":"DROP TABLE things","kind":"dml"}`
	rec := h.do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", body, admin())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "ddl") {
		t.Fatalf("a mismatched kind must be refused: %d %s", rec.Code, rec.Body.String())
	}
}
