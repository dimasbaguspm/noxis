// Package sqlite persists the connection registry (SQLite via modernc,
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

// Satisfiers: the store implements both repository seams.
var (
	_ interfaces.ConnectionRepository = (*Store)(nil)
	_ interfaces.ChangeRepository     = (*Store)(nil)
)

// Open opens the store and verifies it answers a ping. It is deployable
// with zero host-side files: the parent directory is created when missing
// and sqlite creates the database file itself on first open.
func Open(path string) (*Store, error) {
	// The classic bind-mount trap: docker creates a DIRECTORY at a bind
	// target that doesn't exist yet, and sqlite cannot open a directory
	// ("unable to open database file (14)"). Name the failure instead of
	// crash-looping on it.
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	return s, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS changes (
  id          TEXT PRIMARY KEY,
  connection  TEXT NOT NULL,
  notes       TEXT NOT NULL,
  query       TEXT NOT NULL,
  rollback    TEXT NOT NULL,
  status      TEXT NOT NULL,
  checks      TEXT NOT NULL DEFAULT '[]',
  created_at  INTEGER NOT NULL,
  approved_at INTEGER NOT NULL DEFAULT 0,
  applied_at  INTEGER NOT NULL DEFAULT 0
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
);`

func (s *Store) Close() { _ = s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
