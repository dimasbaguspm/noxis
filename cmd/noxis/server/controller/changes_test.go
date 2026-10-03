package controller

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/services/sql"
)

// The seam pays off here: the whole change workflow (submit → validate →
// approve → apply, including its guards) runs against fakes — no database,
// no HTTP.

type fakeConns struct {
	conns map[string]entity.Connection
}

func (f *fakeConns) List(context.Context) ([]entity.Connection, error) {
	out := make([]entity.Connection, 0, len(f.conns))
	for _, c := range f.conns {
		out = append(out, c)
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

func newDeps(sq interfaces.SQLModule) (Deps, *fakeChanges, *fakeConns) {
	conns := &fakeConns{conns: map[string]entity.Connection{
		"demo": {Name: "demo", Driver: constants.DriverPostgres, Host: "db", Port: 5432, User: "postgres", Database: "demo"},
	}}
	changes := &fakeChanges{changes: map[string]entity.Change{}}
	return Deps{
		Connections: conns,
		Changes:     changes,
		SQL:         sq,
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, changes, conns
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
	d, _, _ := newDeps(&fakeSQL{})
	ctx := context.Background()

	submitted, err := d.submitChange(ctx, sampleChange())
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
	if _, err := d.apply(ctx, submitted.ID); err == nil {
		t.Fatal("apply must require approval")
	}

	approved, err := d.approve(ctx, submitted.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != constants.ChangeApproved || approved.ApprovedAt == 0 {
		t.Fatalf("approve result: %+v", approved)
	}

	applied, err := d.apply(ctx, submitted.ID)
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
	d, _, _ := newDeps(&fakeSQL{writeErr: errors.New(`statement "SELECT" not allowed here`)})
	bad, err := d.submitChange(ctx, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if bad.Status != constants.ChangeFailed {
		t.Fatalf("status=%q want failed", bad.Status)
	}
	if _, err := d.approve(ctx, bad.ID); err == nil {
		t.Fatal("failed change must not be approvable")
	}

	// a dry-run failure also lands as failed with the reason recorded
	d2, _, _ := newDeps(&fakeSQL{validateErr: errors.New("rollback statement 1: syntax error")})
	failed, err := d2.submitChange(ctx, sampleChange())
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
	d3, _, _ := newDeps(&fakeSQL{validateErr: sql.ErrUnverifiable})
	unv, err := d3.submitChange(ctx, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if unv.Status != constants.ChangeUnverified {
		t.Fatalf("status=%q want unverified", unv.Status)
	}
	if _, err := d3.approve(ctx, unv.ID); err != nil {
		t.Fatalf("unverified change must be approvable: %v", err)
	}
}

func TestApplyFailureIsRecorded(t *testing.T) {
	d, changes, _ := newDeps(&fakeSQL{applyErr: errors.New("statement 1: duplicate key")})
	ctx := context.Background()
	ch, err := d.submitChange(ctx, sampleChange())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := d.approve(ctx, ch.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := d.apply(ctx, ch.ID); err == nil {
		t.Fatal("apply must surface the failure")
	}
	stored, ok := changes.changes[ch.ID]
	if !ok || stored.Status != constants.ChangeFailed {
		t.Fatalf("failure must be recorded on the change: %+v", stored)
	}
	last := stored.Checks[len(stored.Checks)-1]
	if last.OK || !strings.Contains(last.Error, "duplicate key") {
		t.Fatalf("apply check: %+v", last)
	}
}

func TestSecretsStayRedacted(t *testing.T) {
	d, _, conns := newDeps(&fakeSQL{})
	ctx := context.Background()
	c, _ := conns.Get(ctx, "demo")
	c.Password = "supersecret"
	_ = conns.Save(ctx, *c)

	got, err := d.connection(ctx, "demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Password != "" {
		t.Fatalf("password leaked: %q", got.Password)
	}
	list, err := d.connections(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, x := range list {
		if x.Password != "" {
			t.Fatalf("password leaked in list: %q", x.Password)
		}
	}
	if raw, err := d.raw(ctx, "demo"); err != nil || raw.Password != "supersecret" {
		t.Fatalf("internal path must still see the password: %v", err)
	}
}

// TestHandlersRespond exercises the HTTP path (handle → span → JSON): a
// self-calling span closure once turned this into a stack overflow.
func TestHandlersRespond(t *testing.T) {
	d, _, _ := newDeps(&fakeSQL{})
	h := WithDeps(http.HandlerFunc(ListConnections), d)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/connections", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"connections"`) {
		t.Fatalf("body: %s", rec.Body.String())
	}
}
