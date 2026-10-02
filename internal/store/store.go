// Package store is the master's single source of truth: desired state per
// node, reported statuses, and an append-only changelog. SQLite, pure Go
// driver (modernc) so cross-compilation stays cgo-free.
package store

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type StatusRow struct {
	Node   string
	Body   string
	TS     int64
}

type Store struct {
	db  *sql.DB
	mu  sync.Mutex
	sub map[string]map[chan struct{}]struct{}
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1) // sqlite: one writer
	s := &Store{db: db, sub: map[string]map[chan struct{}]struct{}{}}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	return s, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS specs (
  node       TEXT PRIMARY KEY,
  body       TEXT NOT NULL,
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

func (s *Store) Close() error { return s.db.Close() }

// WriteSpec upserts desired state for a node, logs it, and wakes watchers.
func (s *Store) WriteSpec(node, body string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO specs(node, body, updated_at) VALUES(?,?,?)
		ON CONFLICT(node) DO UPDATE SET body=excluded.body, updated_at=excluded.updated_at`,
		node, body, time.Now().Unix()); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO changelog(node, event, at) VALUES(?,?,?)`,
		node, "set_desired_state", time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.Notify(node)
	return nil
}

func (s *Store) GetSpec(node string) (string, bool, error) {
	var body string
	err := s.db.QueryRow(`SELECT body FROM specs WHERE node=?`, node).Scan(&body)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return body, true, nil
}

func (s *Store) ReportStatus(node, body string) error {
	_, err := s.db.Exec(`INSERT INTO status(node, body, ts) VALUES(?,?,?)
		ON CONFLICT(node) DO UPDATE SET body=excluded.body, ts=excluded.ts`,
		node, body, time.Now().Unix())
	return err
}

func (s *Store) ListStatus() ([]StatusRow, error) {
	rows, err := s.db.Query(`SELECT node, body, ts FROM status ORDER BY node`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatusRow
	for rows.Next() {
		var r StatusRow
		if err := rows.Scan(&r.Node, &r.Body, &r.TS); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Subscribe returns a coalescing signal channel: a tick means "re-read spec".
func (s *Store) Subscribe(node string) chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub[node] == nil {
		s.sub[node] = map[chan struct{}]struct{}{}
	}
	s.sub[node][ch] = struct{}{}
	return ch
}

func (s *Store) Unsubscribe(node string, ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.sub[node]; m != nil {
		delete(m, ch)
	}
}

// Notify wakes all watchers of a node. Non-blocking; a busy watcher gets one
// pending tick, coalescing bursts.
func (s *Store) Notify(node string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.sub[node] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}