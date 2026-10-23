package sql

import (
	"context"
	stdsql "database/sql"
	"path/filepath"
	"testing"

	"noxis/internal/entity"
)

// newSQLiteFile creates a file-backed SQLite database with a small table and
// returns a connection describing it.
func newSQLiteFile(t *testing.T) entity.Connection {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := stdsql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE people (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO people (name) VALUES ('ada'), ('grace');`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return entity.Connection{Name: "app", Driver: "sqlite", Database: path}
}

func TestSQLiteDriverAndDSN(t *testing.T) {
	if _, err := sqlDriver("sqlite"); err != nil {
		t.Fatalf("sqlDriver(sqlite): %v", err)
	}
	got, err := dsn(entity.Connection{Driver: "sqlite", Database: "/tmp/app.db"})
	if err != nil || got != "/tmp/app.db" {
		t.Fatalf("dsn=%q err=%v", got, err)
	}
	got, err = dsn(entity.Connection{Driver: "sqlite", Database: "/tmp/app.db", Params: map[string]string{"_pragma": "busy_timeout(5000)"}})
	if err != nil || got != "file:/tmp/app.db?_pragma=busy_timeout%285000%29" {
		t.Fatalf("dsn with params=%q err=%v", got, err)
	}
}

func TestSQLiteTablesAndRows(t *testing.T) {
	conn := newSQLiteFile(t)
	u := New()
	ctx := context.Background()

	tables, err := u.Tables(ctx, conn)
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	if len(tables) != 1 || tables[0].Name != "people" || tables[0].Schema != "main" {
		t.Fatalf("tables=%+v", tables)
	}
	if len(tables[0].Columns) != 2 || tables[0].Columns[1].Name != "name" || tables[0].Columns[1].Nullable {
		t.Fatalf("columns=%+v", tables[0].Columns)
	}

	res, err := u.Rows(ctx, conn, "main", "people", 10)
	if err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("rows=%+v", res.Rows)
	}
}

// TestSQLiteValidateChangeRollsBack proves the SQLite dry run exercises the live
// schema inside a transaction and leaves it untouched.
func TestSQLiteValidateChangeRollsBack(t *testing.T) {
	conn := newSQLiteFile(t)
	u := New()
	ctx := context.Background()

	if _, err := u.ValidateChange(ctx, conn,
		"INSERT INTO people (name) VALUES ('linus')",
		"DELETE FROM people WHERE name = 'linus'"); err != nil {
		t.Fatalf("validate: %v", err)
	}

	res, err := u.Rows(ctx, conn, "main", "people", 10)
	if err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("dry run leaked rows: %+v", res.Rows)
	}
}

// TestSQLiteOperations covers the operations the compatibility suite runs on the
// server drivers: apply, dry-run, analyze, and the statement-kind guards.
func TestSQLiteOperations(t *testing.T) {
	conn := newSQLiteFile(t)
	u := New()
	ctx := context.Background()

	if err := u.VerifyWrite("INSERT INTO people (name) VALUES ('linus')"); err != nil {
		t.Fatalf("a write must pass the write guard: %v", err)
	}
	if err := u.VerifyWrite("SELECT * FROM people"); err == nil {
		t.Fatal("a read must not pass the write guard")
	}
	if _, err := u.Query(ctx, conn, "INSERT INTO people (name) VALUES ('x')"); err == nil {
		t.Fatal("the read path must refuse DML")
	}
	if _, err := u.Alter(ctx, conn, "SELECT * FROM people"); err == nil {
		t.Fatal("the write path must refuse a read")
	}

	if _, err := u.Apply(ctx, conn, "INSERT INTO people (name) VALUES ('linus')"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// a dry run rolls back, so the count stays at three
	if _, err := u.DryRun(ctx, conn, "INSERT INTO people (name) VALUES ('dry')"); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	res, err := u.Rows(ctx, conn, "main", "people", 10)
	if err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Fatalf("rows=%d want 3 (apply committed once, dry run rolled back)", len(res.Rows))
	}
	if _, err := u.Analyze(ctx, conn, "people"); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if _, err := u.Rows(ctx, conn, "main", "people; DROP TABLE people", 10); err == nil {
		t.Fatal("an identifier that is not a plain name must be refused")
	}
}
