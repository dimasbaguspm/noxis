package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const connColumns = `id, name, driver, host, port, user, password, database, params, created_at, created_by`

// Save upserts a connection (created_at is set on first insert). The id is generated when missing and never overwritten on update (name is the key).
func (s *Store) Save(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	params, err := json.Marshal(c.Params)
	if err != nil {
		return fmt.Errorf("encode params: %w", err)
	}
	password, err := s.secret.Encrypt(c.Password)
	if err != nil {
		return fmt.Errorf("encrypt password for %s: %w", c.Name, err)
	}
	if c.ID == "" {
		c.ID = idgen.UUID()
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO connections (`+connColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			driver=excluded.driver, host=excluded.host, port=excluded.port,
			user=excluded.user, password=excluded.password, database=excluded.database,
			params=excluded.params, created_by=excluded.created_by`,
		c.ID, c.Name, c.Driver, c.Host, c.Port, c.User, password, c.Database, string(params), time.Now().Unix(), c.CreatedBy); err != nil {
		return fmt.Errorf("save connection %s: %w", c.Name, err)
	}
	return nil
}

// List returns every connection (passwords included — callers redact).
func (s *Store) List(ctx context.Context) (out []entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return queryRows(ctx, s.db, `SELECT `+connColumns+` FROM connections ORDER BY name`, s.scanConnection)
}

// Get returns one connection, or constants.ErrNotFound.
func (s *Store) Get(ctx context.Context, name string) (c *entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return s.getOne(ctx, `SELECT `+connColumns+` FROM connections WHERE name = ?`, name)
}

// GetByID returns one connection by its stable id, or constants.ErrNotFound.
func (s *Store) GetByID(ctx context.Context, id string) (c *entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return s.getOne(ctx, `SELECT `+connColumns+` FROM connections WHERE id = ?`, id)
}

func (s *Store) getOne(ctx context.Context, q string, arg any) (*entity.Connection, error) {
	conn, err := s.scanConnection(s.db.QueryRowContext(ctx, q, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("connection %v: %w", arg, constants.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// Delete removes a connection and its changes atomically (orphaned changes would point at a database nobody can reach).
func (s *Store) Delete(ctx context.Context, name string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return s.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM changes WHERE connection = ?`, name); err != nil {
			return fmt.Errorf("delete changes of %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM connection_grants WHERE connection = ?`, name); err != nil {
			return fmt.Errorf("delete grants of %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM connections WHERE name = ?`, name); err != nil {
			return fmt.Errorf("delete connection %s: %w", name, err)
		}
		return nil
	})
}

func (s *Store) scanConnection(row scanner) (entity.Connection, error) {
	var (
		c      entity.Connection
		params string
	)
	if err := row.Scan(&c.ID, &c.Name, &c.Driver, &c.Host, &c.Port, &c.User, &c.Password, &c.Database, &params, &c.CreatedAt, &c.CreatedBy); err != nil {
		return c, err
	}
	plain, err := s.secret.Decrypt(c.Password)
	if err != nil {
		return c, fmt.Errorf("connection %s: %w", c.Name, err)
	}
	c.Password = plain
	if params != "" {
		if err := json.Unmarshal([]byte(params), &c.Params); err != nil {
			return c, fmt.Errorf("decode params for %s: %w", c.Name, err)
		}
	}
	return c, nil
}
