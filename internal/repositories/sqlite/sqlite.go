// Package sqlite persists connections and changes (SQLite via modernc,
// cgo-free) — the app's only state.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"noxis/internal/interfaces"

	// modernc SQLite driver: pure-Go, cgo-free (interface is database/sql).
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

// Satisfiers.
var (
	_ interfaces.ConnectionRepository = (*Store)(nil)
	_ interfaces.ChangeRepository     = (*Store)(nil)
	_ interfaces.GrantRepository      = (*Store)(nil)
)

// Open opens the store and verifies it answers a ping. The parent directory
// is created when missing and sqlite creates the file itself, so the app is
// deployable with zero host-side files.
func Open(ctx context.Context, path string) (*Store, error) {
	// The bind-mount trap: docker creates a DIRECTORY at a missing bind
	// target, and sqlite cannot open a directory ("unable to open database
	// file (14)"). Name it instead of crash-looping.
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		return nil, fmt.Errorf("open %s: is a directory — bind-mount a folder (named volume), not a file", path)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("open %s: mkdir: %w", path, err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	// pre-existing volumes: add columns introduced after the first release
	for _, ddl := range migrations {
		_, _ = db.Exec(ddl)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return s, nil
}

// migrations are applied best-effort; duplicates (fresh databases) error and
// are ignored.
var migrations = []string{
	`ALTER TABLE changes ADD COLUMN kind TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE changes ADD COLUMN rejected_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN est_duration_ms INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN est_affected INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN applied_affected INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN applied_duration_ms INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN affected_tables TEXT NOT NULL DEFAULT '[]'`,
	`ALTER TABLE changes ADD COLUMN affected_columns TEXT NOT NULL DEFAULT '[]'`,
	`ALTER TABLE changes ADD COLUMN scheduled_at INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE changes ADD COLUMN batch_id TEXT NOT NULL DEFAULT ''`,
}

const schema = `
CREATE TABLE IF NOT EXISTS changes (
  id          TEXT PRIMARY KEY,
  connection  TEXT NOT NULL,
  notes       TEXT NOT NULL,
  query       TEXT NOT NULL,
  rollback    TEXT NOT NULL,
  kind        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL,
  checks      TEXT NOT NULL DEFAULT '[]',
  created_at  INTEGER NOT NULL,
  approved_at INTEGER NOT NULL DEFAULT 0,
  applied_at  INTEGER NOT NULL DEFAULT 0,
  rejected_at INTEGER NOT NULL DEFAULT 0,
  est_duration_ms INTEGER NOT NULL DEFAULT 0,
  est_affected    INTEGER NOT NULL DEFAULT 0,
  applied_affected   INTEGER NOT NULL DEFAULT 0,
  applied_duration_ms INTEGER NOT NULL DEFAULT 0,
  scheduled_at INTEGER NOT NULL DEFAULT 0,
  affected_tables  TEXT NOT NULL DEFAULT '[]',
  affected_columns TEXT NOT NULL DEFAULT '[]',
  batch_id    TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS connections (
  name       TEXT PRIMARY KEY,
  driver     TEXT NOT NULL,
  host       TEXT NOT NULL,
  port       INTEGER NOT NULL,
  user       TEXT NOT NULL,
  password   TEXT NOT NULL DEFAULT '',
  database   TEXT NOT NULL,
  params     TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS connection_grants (
  connection TEXT NOT NULL,
  group_name TEXT NOT NULL,
  level      TEXT NOT NULL,
  PRIMARY KEY (connection, group_name)
);`

func (s *Store) Close() { _ = s.db.Close() }

// scanner is the shared row/rows interface.
type scanner interface{ Scan(dest ...any) error }

// queryRows runs a query and scans every row with scan (shared by the list
// methods, so the iteration lives in one place).
func queryRows[T any](ctx context.Context, db *sql.DB, q string, scan func(scanner) (T, error)) ([]T, error) {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// WithTx runs fn in a transaction: commit on success, rollback otherwise.
func (s *Store) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
