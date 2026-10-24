package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// recordEvent appends an audit row in the caller's transaction so the event and grant commit together.
func recordEvent(ctx context.Context, tx *sql.Tx, e entity.AccessEvent) error {
	if e.ID == "" {
		e.ID = idgen.UUID()
	}
	if e.At == 0 {
		e.At = time.Now().Unix()
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO access_events (id, at, actor, action, subject, connection, level, request_id, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.At, e.Actor, e.Action, e.Subject, e.Connection, e.Level, e.RequestID, e.Note); err != nil {
		return fmt.Errorf("record access event: %w", err)
	}
	return nil
}

// ListAccessEvents returns the audit trail for a subject: events about them or performed by them, newest first.
func (s *Store) ListAccessEvents(ctx context.Context, subject string, limit int) (out []entity.AccessEvent, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, at, actor, action, subject, connection, level, request_id, note
		FROM access_events WHERE subject = ? OR actor = ? ORDER BY at DESC LIMIT ?`,
		subject, subject, limit)
	if err != nil {
		return nil, fmt.Errorf("list access events %s: %w", subject, err)
	}
	defer rows.Close()
	for rows.Next() {
		var e entity.AccessEvent
		if err := rows.Scan(&e.ID, &e.At, &e.Actor, &e.Action, &e.Subject, &e.Connection, &e.Level, &e.RequestID, &e.Note); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
