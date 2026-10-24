package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const requestColumns = `id, subject, connection, level, reason, status, requested_at, decided_by, decided_at, decision_note, expires_at`

func scanRequest(row scanner) (*entity.AccessRequest, error) {
	r := &entity.AccessRequest{}
	if err := row.Scan(&r.ID, &r.Subject, &r.Connection, &r.Level, &r.Reason, &r.Status,
		&r.RequestedAt, &r.DecidedBy, &r.DecidedAt, &r.DecisionNote, &r.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	return r, nil
}

// CreateAccessRequest stores a pending request and records the "requested" event in the same transaction.
func (s *Store) CreateAccessRequest(ctx context.Context, r entity.AccessRequest) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create access request: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO access_requests (`+requestColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Subject, r.Connection, r.Level, r.Reason, r.Status, r.RequestedAt,
		r.DecidedBy, r.DecidedAt, r.DecisionNote, r.ExpiresAt); err != nil {
		return fmt.Errorf("create access request: %w", err)
	}
	if err := recordEvent(ctx, tx, entity.AccessEvent{
		Action: constants.AccessRequested, Actor: r.Subject, Subject: r.Subject,
		Connection: r.Connection, Level: r.Level, RequestID: r.ID, Note: r.Reason,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

// PendingRequestExists reports whether the subject already has an open request for the connection.
func (s *Store) PendingRequestExists(ctx context.Context, subject, connection string) (open bool, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM access_requests WHERE subject = ? AND connection = ? AND status = ? LIMIT 1`,
		subject, connection, entity.RequestPending).Scan(&open); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("pending request: %w", err)
	}
	return open, nil
}

// GetAccessRequest returns one request by id.
func (s *Store) GetAccessRequest(ctx context.Context, id string) (r *entity.AccessRequest, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return scanRequest(s.db.QueryRowContext(ctx, `SELECT `+requestColumns+` FROM access_requests WHERE id = ?`, id))
}

// ListAccessRequests returns the subject's own requests, newest first.
func (s *Store) ListAccessRequests(ctx context.Context, subject string) (out []entity.AccessRequest, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+requestColumns+` FROM access_requests WHERE subject = ? ORDER BY requested_at DESC`, subject)
	if err != nil {
		return nil, fmt.Errorf("list access requests: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ListPendingRequests returns pending requests, optionally restricted to the given connections.
func (s *Store) ListPendingRequests(ctx context.Context, connections []string) (out []entity.AccessRequest, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	query := `SELECT ` + requestColumns + ` FROM access_requests WHERE status = ?`
	args := []any{entity.RequestPending}
	if connections != nil {
		if len(connections) == 0 {
			return nil, nil
		}
		query += ` AND connection IN (` + strings.TrimSuffix(strings.Repeat("?,", len(connections)), ",") + `)`
		for _, c := range connections {
			args = append(args, c)
		}
	}
	query += ` ORDER BY requested_at`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending requests: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// SetAccessRequestDecision closes a pending request and records the decision event.
func (s *Store) SetAccessRequestDecision(ctx context.Context, id, status, decidedBy string, decidedAt int64, note string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("decide request: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	r, err := scanRequest(tx.QueryRowContext(ctx, `SELECT `+requestColumns+` FROM access_requests WHERE id = ?`, id))
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE access_requests SET status = ?, decided_by = ?, decided_at = ?, decision_note = ?
		WHERE id = ? AND status = ?`,
		status, decidedBy, decidedAt, note, id, entity.RequestPending)
	if err != nil {
		return fmt.Errorf("decide request: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return fmt.Errorf("request %s is no longer pending", id)
	}
	if err := recordEvent(ctx, tx, entity.AccessEvent{
		Action: status, Actor: decidedBy, Subject: r.Subject,
		Connection: r.Connection, Level: r.Level, RequestID: id, Note: note, At: decidedAt,
	}); err != nil {
		return err
	}
	return tx.Commit()
}
