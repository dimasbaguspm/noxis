package sql

import (
	"context"
	stdsql "database/sql"
	"path/filepath"
	"testing"
)

// SQLite exposes indexes through pragmas; a catalog index and a unique constraint both reach the table.
func TestListTablesReadsSQLiteIndexes(t *testing.T) {
	db, err := stdsql.Open("sqlite", filepath.Join(t.TempDir(), "schema.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, stmt := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL UNIQUE, name TEXT)`,
		`CREATE INDEX users_name ON users(name)`,
	} {
		if _, err := db.ExecContext(context.Background(), stmt); err != nil {
			t.Fatal(err)
		}
	}

	tables, err := listTables(context.Background(), db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 {
		t.Fatalf("tables = %+v", tables)
	}
	if tables[0].EstimatedRows != nil {
		t.Fatalf("SQLite has no estimate, got %d", *tables[0].EstimatedRows)
	}
	var named, unique bool
	for _, idx := range tables[0].Indexes {
		if idx.Name == "users_name" && len(idx.Columns) == 1 && idx.Columns[0] == "name" && !idx.Unique {
			named = true
		}
		if idx.Unique && len(idx.Columns) == 1 && idx.Columns[0] == "email" {
			unique = true
		}
	}
	if !named || !unique {
		t.Fatalf("indexes = %+v", tables[0].Indexes)
	}
}
