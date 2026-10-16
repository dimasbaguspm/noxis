package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"noxis/cmd/noxis/server/deps"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/modules/sql"
)

// The seam pays off here: the whole change workflow (submit → validate →
// approve → apply, including its guards) runs against fakes — no database,
// no HTTP.

type fakeConns struct {
	conns map[string]entity.Connection
}

func (f *fakeConns) List(context.Context) ([]entity.Connection, error) {
	out := make([]entity.Connection, 0, len(f.conns))
	for name := range f.conns {
		out = append(out, f.conns[name])
	}
	return out, nil
}

func (f *fakeConns) Get(_ context.Context, name string) (*entity.Connection, error) {
	c, ok := f.conns[name]
	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return &c, nil
}

func (f *fakeConns) GetByID(_ context.Context, id string) (*entity.Connection, error) {
	for name := range f.conns {
		if f.conns[name].ID == id {
			c := f.conns[name]
			return &c, nil
		}
	}
	return nil, interfaces.ErrNotFound
}

func (f *fakeConns) Save(_ context.Context, c entity.Connection) error {
	f.conns[c.Name] = c
	return nil
}

func (f *fakeConns) Delete(_ context.Context, name string) error {
	delete(f.conns, name)
	return nil
}

type fakeChanges struct {
	changes map[string]entity.Change
}

func (f *fakeChanges) ListChanges(context.Context) ([]entity.Change, error) {
	out := make([]entity.Change, 0, len(f.changes))
	for id := range f.changes {
		out = append(out, f.changes[id])
	}
	return out, nil
}

func (f *fakeChanges) GetChange(_ context.Context, id string) (*entity.Change, error) {
	c, ok := f.changes[id]
	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return &c, nil
}

func (f *fakeChanges) SaveChange(_ context.Context, c entity.Change) error {
	f.changes[c.ID] = c
	return nil
}

func (f *fakeChanges) DeleteChange(_ context.Context, id string) error {
	delete(f.changes, id)
	return nil
}

// fakeSQL embeds the seam for defaults and overrides what the flow touches.
type fakeSQL struct {
	interfaces.SQLModule
	validateErr error
	applyErr    error
	writeErr    error
}

func (f *fakeSQL) Supports(driver string) bool {
	return driver == constants.DriverPostgres || driver == constants.DriverMySQL
}

func (f *fakeSQL) Family() string { return constants.FamilySQL }

func (f *fakeSQL) VerifyWrite(string) error { return f.writeErr }

func (f *fakeSQL) Kind(string) string { return constants.KindDDL }

func (f *fakeSQL) ValidateChange(context.Context, entity.Connection, string, string) (interfaces.Result, error) {
	if f.validateErr != nil {
		return interfaces.Result{}, f.validateErr
	}
	return interfaces.Result{Message: "query and rollback both valid (rolled back)"}, nil
}

func (f *fakeSQL) Apply(context.Context, entity.Connection, string) (interfaces.Result, error) {
	if f.applyErr != nil {
		return interfaces.Result{}, f.applyErr
	}
	return interfaces.Result{Message: "applied: 1 statement(s)"}, nil
}

// testBase is the explicit dependency bundle the tests hand to the gateway
// constructors; every gateway field is filled from it.
type testBase struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Permissions interfaces.PermissionRepository
	Modules     map[string]interfaces.Module
}

// newTestBase builds the dependency bundle against fakes; newTestDeps adds
// the transport-level deps around it. Tests then wire the workflow seam they
// exercise.
func newTestBase(sq interfaces.SQLModule) (testBase, *fakeChanges, *fakeConns) {
	conns := &fakeConns{conns: map[string]entity.Connection{
		"demo": {Name: "demo", Driver: constants.DriverPostgres, Host: "db", Port: 5432, User: "postgres", Database: "demo"},
	}}
	changes := &fakeChanges{changes: map[string]entity.Change{}}
	return testBase{
		Connections: conns,
		Changes:     changes,
		Permissions: &fakePermissions{permissions: map[string]entity.Permission{}},
		Modules:     map[string]interfaces.Module{constants.FamilySQL: sq},
	}, changes, conns
}

// newTestDeps binds the use cases against fresh fakes, exactly like the
// initiator does at boot; the fakes are returned for store inspection.
func newTestDeps(sq interfaces.SQLModule) (deps.Uses, *fakeChanges, *fakeConns) {
	base, changes, conns := newTestBase(sq)
	return deps.Init(deps.Deps{
		Connections: base.Connections,
		Changes:     base.Changes,
		Permissions: base.Permissions,
		Modules:     modules.New(base.Modules, nil),
		Log:         discardLog(),
	}), changes, conns
}

func sampleChange() entity.Change {
	return entity.Change{
		Connection: "demo",
		Notes:      "add things",
		Query:      "CREATE TABLE things (id INT)",
		Rollback:   "DROP TABLE things",
	}
}

func TestChangeWorkflow(t *testing.T) {
	sq := &fakeSQL{}
	d, _, _ := newTestDeps(sq)
	ctx := modules.WithRegistry(context.Background(), d.Modules)

	submitted, err := d.SubmitChange.Serve(ctx, sq, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if submitted.Status != constants.ChangeValidated || submitted.ID == "" || len(submitted.Checks) != 2 {
		t.Fatalf("submit result: status=%q id=%q checks=%d", submitted.Status, submitted.ID, len(submitted.Checks))
	}
	if submitted.Kind != constants.KindDDL {
		t.Fatalf("kind=%q want ddl", submitted.Kind)
	}

	// apply before approval is refused
	if _, err := d.Apply.Serve(ctx, sq, submitted.ID); err == nil {
		t.Fatal("apply must require approval")
	}

	approved, err := d.Approve.Serve(ctx, submitted.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != constants.ChangeApproved || approved.ApprovedAt == 0 {
		t.Fatalf("approve result: %+v", approved)
	}

	applied, err := d.Apply.Serve(ctx, sq, submitted.ID)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if applied.Status != constants.ChangeApplied || applied.AppliedAt == 0 {
		t.Fatalf("apply result: %+v", applied)
	}
}

func TestChangeValidationFailures(t *testing.T) {
	ctx := context.Background()

	// a read on the change path is refused up front
	sq := &fakeSQL{writeErr: errors.New(`statement "SELECT" not allowed here`)}
	d, _, _ := newTestDeps(sq)
	bad, err := d.SubmitChange.Serve(ctx, sq, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if bad.Status != constants.ChangeFailed {
		t.Fatalf("status=%q want failed", bad.Status)
	}
	if _, err := d.Approve.Serve(ctx, bad.ID); err == nil {
		t.Fatal("failed change must not be approvable")
	}

	// a dry-run failure also lands as failed with the reason recorded
	sq2 := &fakeSQL{validateErr: errors.New("rollback statement 1: syntax error")}
	d2, _, _ := newTestDeps(sq2)
	failed, err := d2.SubmitChange.Serve(ctx, sq2, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if failed.Status != constants.ChangeFailed {
		t.Fatalf("status=%q want failed", failed.Status)
	}
	if len(failed.Checks) < 2 || failed.Checks[1].OK || !strings.Contains(failed.Checks[1].Error, "syntax error") {
		t.Fatalf("checks: %+v", failed.Checks)
	}

	// mysql DDL cannot be dry-run: unverified, still approvable
	sq3 := &fakeSQL{validateErr: sql.ErrUnverifiable}
	d3, _, _ := newTestDeps(sq3)
	unv, err := d3.SubmitChange.Serve(ctx, sq3, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if unv.Status != constants.ChangeUnverified {
		t.Fatalf("status=%q want unverified", unv.Status)
	}
	if _, err := d3.Approve.Serve(ctx, unv.ID); err != nil {
		t.Fatalf("unverified change must be approvable: %v", err)
	}
}

func TestApplyFailureIsRecorded(t *testing.T) {
	sq := &fakeSQL{applyErr: errors.New("statement 1: duplicate key")}
	d, store, _ := newTestDeps(sq)
	ctx := modules.WithRegistry(context.Background(), d.Modules)
	ch, err := d.SubmitChange.Serve(ctx, sq, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := d.Approve.Serve(ctx, ch.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := d.Apply.Serve(ctx, sq, ch.ID); err == nil {
		t.Fatal("apply must surface the failure")
	}
	stored, ok := store.changes[ch.ID]
	if !ok || stored.Status != constants.ChangeFailed {
		t.Fatalf("failure must be recorded on the change: %+v", stored)
	}
	last := stored.Checks[len(stored.Checks)-1]
	if last.OK || !strings.Contains(last.Error, "duplicate key") {
		t.Fatalf("apply check: %+v", last)
	}
}

func TestSecretsStayRedacted(t *testing.T) {
	d, _, conns := newTestDeps(&fakeSQL{})
	mod, _ := d.Modules.Engine(constants.FamilySQL)
	ctx := modules.WithRegistry(context.Background(), d.Modules)
	c, _ := conns.Get(ctx, "demo")
	c.Password = "supersecret"
	_ = conns.Save(ctx, *c)

	got, err := d.Connection.Serve(ctx, mod, "demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Password != "" {
		t.Fatalf("password leaked: %q", got.Password)
	}
	list, err := d.ConnectionList.Serve(ctx, mod)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, x := range list {
		if x.Password != "" {
			t.Fatalf("password leaked in list: %q", x.Password)
		}
	}
	if raw, err := d.Raw.Serve(ctx, mod, "demo", entity.LevelRead); err != nil || raw.Password != "supersecret" {
		t.Fatalf("internal path must still see the password: %v", err)
	}
}

// TestHandlersRespond exercises the HTTP path (handle → span → JSON): a
// self-calling span closure once turned this into a stack overflow.
func TestHandlersRespond(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /external/v1/engine/{engine}/connections", ListConnections)

	rec := httptest.NewRecorder()
	depsHandler(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/external/v1/engine/sql/connections", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"connections"`) {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

// TestEngineGating locks the generic route contract: an unknown engine and a
// sub-resource the engine does not serve are 404, never routed elsewhere.
func TestEngineGating(t *testing.T) {
	d, _, _ := newTestDeps(&fakeSQL{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /external/v1/engine/{engine}/connections", ListConnections)
	mux.HandleFunc("GET /external/v1/engine/{engine}/connections/{name}/keys", ListKeys)

	cases := []struct {
		path string
		want int
	}{
		{"/external/v1/engine/nope/connections", http.StatusNotFound},
		{"/external/v1/engine/sql/connections/demo/keys", http.StatusNotFound},
		{"/external/v1/engine/sql/connections", http.StatusOK},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		depsHandler(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, http.NoBody))
		if rec.Code != tc.want {
			t.Fatalf("%s: status=%d want=%d body=%s", tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}

// TestScheduledChangeWaitsForItsSlot: a change scheduled in the future is not
// applicable by hand and is skipped by a tick before its time.
func TestScheduledChangeWaitsForItsSlot(t *testing.T) {
	sq := &fakeSQL{}
	d, store, _ := newTestDeps(sq)
	ctx := modules.WithRegistry(context.Background(), d.Modules)
	in := sampleChange()
	in.ScheduledAt = time.Now().Add(time.Hour).Unix()
	ch, err := d.SubmitChange.Serve(ctx, sq, in)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := d.Approve.Serve(ctx, ch.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := d.Apply.Serve(ctx, sq, ch.ID); err == nil {
		t.Fatal("apply before the scheduled time must be refused")
	}
	res, err := d.Tick.Serve(ctx)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(res.Applied) != 0 || len(res.Failed) != 0 {
		t.Fatalf("future change must not be applied: %+v", res)
	}
	if got := store.changes[ch.ID].Status; got != constants.ChangeApproved {
		t.Fatalf("status=%q want approved", got)
	}
}

// TestTickAppliesDueScheduledChange: an approved change whose time has passed
// is applied by the tick; an immediate change is left for the operator.
func TestTickAppliesDueScheduledChange(t *testing.T) {
	sq := &fakeSQL{}
	d, store, _ := newTestDeps(sq)
	ctx := modules.WithRegistry(context.Background(), d.Modules)
	in := sampleChange()
	in.ScheduledAt = time.Now().Add(-time.Minute).Unix()
	ch, err := d.SubmitChange.Serve(ctx, sq, in)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	immediate, err := d.SubmitChange.Serve(ctx, sq, sampleChange())
	if err != nil {
		t.Fatalf("submit immediate: %v", err)
	}
	for _, id := range []string{ch.ID, immediate.ID} {
		if _, err := d.Approve.Serve(ctx, id); err != nil {
			t.Fatalf("approve %s: %v", id, err)
		}
	}
	res, err := d.Tick.Serve(ctx)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(res.Applied) != 1 || res.Applied[0] != ch.ID {
		t.Fatalf("applied=%v want [%s]", res.Applied, ch.ID)
	}
	if got := store.changes[ch.ID].Status; got != constants.ChangeApplied {
		t.Fatalf("due change status=%q want applied", got)
	}
	if got := store.changes[immediate.ID].Status; got != constants.ChangeApproved {
		t.Fatalf("immediate change status=%q want approved", got)
	}
}

// TestSchedulerEndpoint runs the keyless external scheduler hook.
func TestSchedulerEndpoint(t *testing.T) {
	d, fakeStore, _ := newTestDeps(&fakeSQL{})
	fakeStore.changes["due"] = entity.Change{ID: "due", Kind: constants.KindDDL, Status: constants.ChangeApproved, Connection: "demo"}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/callback/scheduler", PostScheduler)

	for _, path := range []string{
		"/internal/callback/scheduler",
		"/internal/callback/scheduler",
	} {
		req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
		rec := httptest.NewRecorder()
		depsHandler(mux, d).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}

// TestChangeKindGated locks the family guard on the shared change routes: a
// SQL engine URL cannot approve or reject a MongoDB request (direct id), and
// the reverse is guarded by MongoChange.
func TestChangeKindGated(t *testing.T) {
	d, fakeStore, _ := newTestDeps(&fakeSQL{})
	fakeStore.changes["m1"] = entity.Change{ID: "m1", Kind: constants.KindMongo, Status: constants.ChangeValidated, Connection: "mongo"}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /external/v1/engine/{engine}/changes/{id}/approve", ApproveChange)
	mux.HandleFunc("POST /external/v1/engine/{engine}/changes/{id}/reject", RejectChange)

	for _, path := range []string{"/external/v1/engine/sql/changes/m1/approve", "/external/v1/engine/sql/changes/m1/reject"} {
		rec := httptest.NewRecorder()
		depsHandler(mux, d).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, http.NoBody))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d want=400 body=%s", path, rec.Code, rec.Body.String())
		}
	}
}
