// Package sqlite persists the master's slave registry (SQLite via modernc,
// cgo-free). Specs and status live in Kafka + master memory now — the store
// is deliberately just the registry.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

// Open opens the registry and verifies it answers a ping.
func Open(path string) (*Store, error) {
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

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
