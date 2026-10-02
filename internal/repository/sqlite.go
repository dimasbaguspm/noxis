// Package repository implements the interfaces.SpecRepository and
// StatusRepository over SQLite (pure-Go modernc driver, cgo-free for
// cross-compilation). One connection: sqlite serializes writes anyway.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db  *sql.DB
	mu  sync.Mutex
	sub map[string]map[chan struct{}]struct{}
}

func OpenSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	r := &SQLite{db: db, sub: map[string]map[chan struct{}]struct{}{}}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	return r, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS specs (
  node       TEXT PRIMARY KEY,
  body       TEXT NOT NULL,      -- yaml assignment record
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS status (
  node TEXT PRIMARY KEY,
  body TEXT NOT NULL,
  ts   INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS changelog (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  node  TEXT NOT NULL,
  event TEXT NOT NULL,
  at    INTEGER NOT NULL
);`

func (r *SQLite) Close() error { return r.db.Close() }

// Notify wakes all watchers of a node, coalescing bursts into one tick.
func (r *SQLite) Notify(ctx context.Context, node string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.sub[node] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
