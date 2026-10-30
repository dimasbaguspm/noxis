package change

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	"noxis/internal/modules/postgres"
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
	registry := modules.New(
		map[string]interfaces.Module{constants.DriverPostgres: h.sq},
		interfaces.ChangeFlows{constants.DriverPostgres: postgres.NewChanges(h.conns, h.changes, h.sq, nil)},
	)
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

func TestSubmitWithoutModuleLoaderFailsClosed(t *testing.T) {
	h := newHarness(t)
	rec := testkit.Do(testkit.Mux("POST", "/external/v1/engine/{driver}/changes", Submit{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/external/v1/engine/postgres/changes", sampleSQL, admin())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), common.ErrModuleLoaderUnavailable.Error()) {
		t.Fatalf("missing module loader status=%d body=%s", rec.Code, rec.Body.String())
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
