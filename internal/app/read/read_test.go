package read

import (
	"context"
	"errors"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"strings"
	"testing"
	"time"
)

// ---- doubles (handwritten, in the package that uses them) ----

type fakeTargets struct {
	target domain.Target
	err    error
}

func (*fakeTargets) Create(_ context.Context, target domain.Target) (domain.Target, error) {
	return target, nil
}

func (f *fakeTargets) ByRef(_ context.Context, _ string) (domain.Target, error) {
	if f.err != nil {
		return domain.Target{}, f.err
	}
	return f.target, nil
}

func (*fakeTargets) List(_ context.Context, _ ports.TargetFilter) ([]domain.Target, error) {
	return nil, nil
}

func (*fakeTargets) Update(_ context.Context, target domain.Target) (domain.Target, error) {
	return target, nil
}

type fakeSealer struct{ openErr error }

func (*fakeSealer) Enabled() bool { return true }

func (*fakeSealer) Seal(_ string, plain []byte) (string, error) {
	return "sealed:" + string(plain), nil
}

func (f *fakeSealer) Open(_ string, sealed string) ([]byte, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return []byte(strings.TrimPrefix(sealed, "sealed:")), nil
}

// fakeDriver records the password it was handed, which is how the tests prove the plaintext reaches
// the engine and nothing else.
type fakeDriver struct {
	password  string
	statement string
	rows      domain.Rows
	err       error
	databases []string
	tables    []domain.Table
	table     domain.TableDetail
	calls     int
}

func (*fakeDriver) Probe(_ context.Context, _ domain.Target, _ string) (domain.Capabilities, error) {
	return domain.Capabilities{}, nil
}

func (f *fakeDriver) Query(_ context.Context, _ domain.Target, password, statement string, _ int) (domain.Rows, error) {
	f.calls++
	f.password, f.statement = password, statement
	return f.rows, f.err
}

func (f *fakeDriver) Databases(_ context.Context, _ domain.Target, password string) ([]string, error) {
	f.calls++
	f.password = password
	return f.databases, f.err
}

func (f *fakeDriver) Tables(_ context.Context, _ domain.Target, password, _ string) ([]domain.Table, error) {
	f.calls++
	f.password = password
	return f.tables, f.err
}

func (f *fakeDriver) Table(_ context.Context, _ domain.Target, password, _, _ string) (domain.TableDetail, error) {
	f.calls++
	f.password = password
	return f.table, f.err
}

func (*fakeDriver) Rehearse(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Rehearsal, error) {
	return domain.Rehearsal{}, nil
}

func (*fakeDriver) Plan(_ context.Context, _ domain.Target, _ string, _ domain.Change) (domain.Plan, error) {
	return domain.Plan{}, nil
}

func (*fakeDriver) Apply(_ context.Context, _ domain.Target, _ string, _ domain.Change,
	_ func(domain.Progress),
) (domain.Receipt, error) {
	return domain.Receipt{}, nil
}

type fixture struct {
	deps    Deps
	target  domain.Target
	targets *fakeTargets
	sealer  *fakeSealer
	driver  *fakeDriver
}

func newFixture() *fixture {
	target := domain.Target{
		Ref: "0192f0c0-0000-7000-8000-000000000001", Name: "prod", Driver: "mysql",
		Host: "db", Port: 3306, Database: "app", Username: "noxis", Secret: "sealed:pw",
		TLSMode: "preferred", CreatedBy: "ops",
	}
	f := &fixture{
		target:  target,
		targets: &fakeTargets{target: target},
		sealer:  &fakeSealer{},
		driver:  &fakeDriver{},
	}
	f.deps = Deps{Targets: f.targets, Sealer: f.sealer, Driver: f.driver, Timeout: time.Second}
	return f
}

// ---- query ----

// The refusal happens before a connection is opened: a write never reaches an engine, even one that
// would have accepted it.
func TestQueryRefusesAWriteBeforeConnecting(t *testing.T) {
	f := newFixture()

	for _, sql := range []string{"DELETE FROM users", "UPDATE users SET note = 'x'", "SELECT 1; DROP TABLE users"} {
		if _, err := NewQuery(f.deps).Run(context.Background(), QueryInput{Ref: f.target.Ref, SQL: sql}); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("%q: %v, want ErrInvalid", sql, err)
		}
	}
	if f.driver.calls != 0 {
		t.Fatalf("the driver was called %d times for a refused statement", f.driver.calls)
	}
}

// A read that takes locks is refused too: the route is called read for a reason.
func TestQueryRefusesALockingRead(t *testing.T) {
	f := newFixture()
	if _, err := NewQuery(f.deps).Run(context.Background(), QueryInput{
		Ref: f.target.Ref, SQL: "SELECT * FROM users FOR UPDATE",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("locking read: %v, want ErrInvalid", err)
	}
	if f.driver.calls != 0 {
		t.Fatal("a locking read reached the engine")
	}
}

func TestQueryReadsThroughTheTargetPassword(t *testing.T) {
	f := newFixture()
	f.driver.rows = domain.Rows{Columns: []domain.Column{{Name: "id"}}}

	rows, err := NewQuery(f.deps).Run(context.Background(), QueryInput{Ref: f.target.Ref, SQL: "SELECT id FROM users"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows.Columns) != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	if f.driver.password != "pw" {
		t.Fatalf("the driver got password %q, want the opened one", f.driver.password)
	}
	if f.driver.statement != "SELECT id FROM users" {
		t.Fatalf("statement = %q, want it unchanged", f.driver.statement)
	}
}

func TestConnectRefusals(t *testing.T) {
	f := newFixture()
	if _, err := NewQuery(f.deps).Run(context.Background(), QueryInput{Ref: "prod", SQL: "SELECT 1"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("bad reference: %v, want ErrInvalid", err)
	}

	when := time.Now().UTC()
	f.targets.target.ArchivedAt = &when
	if _, err := NewQuery(f.deps).Run(context.Background(), QueryInput{
		Ref: f.target.Ref, SQL: "SELECT 1",
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("archived target: %v, want ErrConflict", err)
	}

	// A password that cannot be opened is a key problem, and saying so is the whole message.
	f = newFixture()
	f.sealer.openErr = errors.New("cipher: message authentication failed")
	if _, err := NewQuery(f.deps).Run(context.Background(), QueryInput{
		Ref: f.target.Ref, SQL: "SELECT 1",
	}); !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("unopenable password: %v, want ErrUnsupported", err)
	}

	// A target with no password at all connects with an empty one.
	f = newFixture()
	f.targets.target.Secret = ""
	if _, err := NewQuery(f.deps).Run(context.Background(), QueryInput{
		Ref: f.target.Ref, SQL: "SELECT 1",
	}); err != nil {
		t.Fatalf("passwordless target: %v", err)
	}
}

// ---- metadata ----

func TestMetadata(t *testing.T) {
	f := newFixture()
	f.driver.databases = []string{"app", "mysql"}
	f.driver.tables = []domain.Table{{Name: "users"}}
	f.driver.table = domain.TableDetail{Table: domain.Table{Name: "users"}}

	databases, err := NewDatabases(f.deps).Run(context.Background(), f.target.Ref)
	if err != nil || len(databases) != 2 {
		t.Fatalf("Databases = %v, %v", databases, err)
	}

	tables, err := NewTables(f.deps).Run(context.Background(), TablesInput{Ref: f.target.Ref, Database: "app"})
	if err != nil || len(tables) != 1 {
		t.Fatalf("Tables = %v, %v", tables, err)
	}

	table, err := NewTable(f.deps).Run(context.Background(), TableInput{
		Ref: f.target.Ref, Database: "app", Table: "users",
	})
	if err != nil || table.Name != "users" {
		t.Fatalf("Table = %+v, %v", table, err)
	}

	// A missing name is a 400 before any connection is attempted.
	if _, err := NewTables(f.deps).Run(context.Background(), TablesInput{Ref: f.target.Ref}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("Tables with no schema: %v, want ErrInvalid", err)
	}
	if _, err := NewTable(f.deps).Run(context.Background(), TableInput{
		Ref: f.target.Ref, Database: "app",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("Table with no table: %v, want ErrInvalid", err)
	}
}
