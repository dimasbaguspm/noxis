package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Save upserts a connection (created_at is set on first insert).
func (s *Store) Save(ctx context.Context, c entity.Connection) error {
	params, err := json.Marshal(c.Params)
	if err != nil {
		return fmt.Errorf("encode params: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO connections (name, driver, host, port, user, password, database, params, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			driver=excluded.driver, host=excluded.host, port=excluded.port,
			user=excluded.user, password=excluded.password, database=excluded.database,
			params=excluded.params`,
		c.Name, c.Driver, c.Host, c.Port, c.User, c.Password, c.Database, string(params), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save connection %s: %w", c.Name, err)
	}
	return nil
}

// List returns every connection (passwords included — callers redact).
func (s *Store) List(ctx context.Context) ([]entity.Connection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, driver, host, port, user, password, database, params, created_at
		FROM connections ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()
	var out []entity.Connection
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get returns one connection or interfaces.ErrNotFound.
func (s *Store) Get(ctx context.Context, name string) (*entity.Connection, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT name, driver, host, port, user, password, database, params, created_at
		FROM connections WHERE name = ?`, name)
	c, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("connection %s: %w", name, interfaces.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Delete removes a connection and its changes, atomically (orphaned changes
// would point at a database nobody can reach).
func (s *Store) Delete(ctx context.Context, name string) error {
	return s.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM changes WHERE connection = ?`, name); err != nil {
			return fmt.Errorf("delete changes of %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM connections WHERE name = ?`, name); err != nil {
			return fmt.Errorf("delete connection %s: %w", name, err)
		}
		return nil
	})
}

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner) (entity.Connection, error) {
	var c entity.Connection
	var params string
	if err := row.Scan(&c.Name, &c.Driver, &c.Host, &c.Port, &c.User, &c.Password, &c.Database, &params, &c.CreatedAt); err != nil {
		return c, err
	}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &c.Params); err != nil {
			return c, fmt.Errorf("decode params for %s: %w", c.Name, err)
		}
	}
	return c, nil
}
