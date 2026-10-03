package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const changeColumns = `id, connection, notes, query, rollback, kind, status, checks, created_at, approved_at, applied_at`

// SaveChange upserts a change.
func (s *Store) SaveChange(ctx context.Context, c entity.Change) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	checks, err := json.Marshal(c.Checks)
	if err != nil {
		return fmt.Errorf("encode checks: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO changes (`+changeColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connection=excluded.connection, notes=excluded.notes, query=excluded.query,
			rollback=excluded.rollback, kind=excluded.kind, status=excluded.status, checks=excluded.checks,
			approved_at=excluded.approved_at, applied_at=excluded.applied_at`,
		c.ID, c.Connection, c.Notes, c.Query, c.Rollback, c.Kind, c.Status, string(checks),
		c.CreatedAt, c.ApprovedAt, c.AppliedAt); err != nil {
		return fmt.Errorf("save change %s: %w", c.ID, err)
	}
	return nil
}

// ListChanges returns changes, newest first.
func (s *Store) ListChanges(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return queryRows(ctx, s.db, `SELECT `+changeColumns+` FROM changes ORDER BY created_at DESC, id`, scanChange)
}

// GetChange returns one change, or interfaces.ErrNotFound.
func (s *Store) GetChange(ctx context.Context, id string) (c *entity.Change, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	row := s.db.QueryRowContext(ctx, `SELECT `+changeColumns+` FROM changes WHERE id = ?`, id)
	ch, err := scanChange(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("change %s: %w", id, interfaces.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// DeleteChange removes a change (reject = delete).
func (s *Store) DeleteChange(ctx context.Context, id string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM changes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete change %s: %w", id, err)
	}
	return nil
}

func scanChange(row scanner) (entity.Change, error) {
	var (
		c      entity.Change
		checks string
	)
	if err := row.Scan(&c.ID, &c.Connection, &c.Notes, &c.Query, &c.Rollback, &c.Kind, &c.Status, &checks,
		&c.CreatedAt, &c.ApprovedAt, &c.AppliedAt); err != nil {
		return c, err
	}
	if checks != "" {
		if err := json.Unmarshal([]byte(checks), &c.Checks); err != nil {
			return c, fmt.Errorf("decode checks for %s: %w", c.ID, err)
		}
	}
	return c, nil
}
