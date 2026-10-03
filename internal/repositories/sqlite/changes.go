package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Save upserts a change.
func (s *Store) SaveChange(ctx context.Context, c entity.Change) error {
	checks, err := json.Marshal(c.Checks)
	if err != nil {
		return fmt.Errorf("encode checks: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO changes (id, connection, notes, query, rollback, status, checks, created_at, approved_at, applied_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			connection=excluded.connection, notes=excluded.notes, query=excluded.query,
			rollback=excluded.rollback, status=excluded.status, checks=excluded.checks,
			approved_at=excluded.approved_at, applied_at=excluded.applied_at`,
		c.ID, c.Connection, c.Notes, c.Query, c.Rollback, c.Status, string(checks),
		c.CreatedAt, c.ApprovedAt, c.AppliedAt)
	if err != nil {
		return fmt.Errorf("save change %s: %w", c.ID, err)
	}
	return nil
}

// ListChanges returns changes, newest first.
func (s *Store) ListChanges(ctx context.Context) ([]entity.Change, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, connection, notes, query, rollback, status, checks, created_at, approved_at, applied_at
		FROM changes ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list changes: %w", err)
	}
	defer rows.Close()
	var out []entity.Change
	for rows.Next() {
		c, err := scanChange(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChange returns one change or interfaces.ErrNotFound.
func (s *Store) GetChange(ctx context.Context, id string) (*entity.Change, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, connection, notes, query, rollback, status, checks, created_at, approved_at, applied_at
		FROM changes WHERE id = ?`, id)
	c, err := scanChange(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("change %s: %w", id, interfaces.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteChange removes a change (reject = delete).
func (s *Store) DeleteChange(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM changes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete change %s: %w", id, err)
	}
	return nil
}

func scanChange(row scanner) (entity.Change, error) {
	var c entity.Change
	var checks string
	if err := row.Scan(&c.ID, &c.Connection, &c.Notes, &c.Query, &c.Rollback, &c.Status, &checks,
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
