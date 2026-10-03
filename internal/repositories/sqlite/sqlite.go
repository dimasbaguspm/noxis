// Package sqlite persists the master's slave registry (SQLite via modernc,
// cgo-free). Specs and status live in Kafka + master memory now — the store
// is deliberately just the registry.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"noxis/internal/interfaces"

	// modernc SQLite driver: pure-Go, cgo-free (interface is database/sql).
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

// Satisfier: the registry store implements the node repository seam.
var _ interfaces.NodeRepository = (*Store)(nil)

// Open opens the registry and verifies it answers a ping.
func Open(path string) (*Store, error) {
	// The classic bind-mount trap: docker creates a DIRECTORY at a bind
	// target that doesn't exist yet, and sqlite cannot open a directory
	// ("unable to open database file (14)"). Name the failure instead of
	// crash-looping on it.
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		return nil, fmt.Errorf("open %s: is a directory — bind-mount a folder (named volume), not a file", path)
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
CREATE TABLE IF NOT EXISTS slaves (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  ip            TEXT NOT NULL DEFAULT '',
  registered_at INTEGER NOT NULL
);`

func (s *Store) Close() { _ = s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
