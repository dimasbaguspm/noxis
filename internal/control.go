// Package control is the app's operations core: connections, the SQL module,
// and the change workflow (submit → validate → approve → apply).
package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	sqlops "noxis/internal/sql"
)

// Control carries the connection registry, the change registry, and the SQL
// module they run through.
type Control struct {
	conns   interfaces.ConnectionRepository
	changes interfaces.ChangeRepository
	db      *sqlops.DB
	log     *slog.Logger
}

// New wires the stores and the SQL module.
func New(conns interfaces.ConnectionRepository, changes interfaces.ChangeRepository, log *slog.Logger) *Control {
	return &Control{conns: conns, changes: changes, db: sqlops.New(), log: log}
}

// Connections lists the registry, passwords redacted.
func (c *Control) Connections(ctx context.Context) ([]entity.Connection, error) {
	list, err := c.conns.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.Connection, 0, len(list))
	for _, x := range list {
		out = append(out, x.Redacted())
	}
	return out, nil
}

// Connection returns one redacted connection.
func (c *Control) Connection(ctx context.Context, name string) (entity.Connection, error) {
	x, err := c.conns.Get(ctx, name)
	if err != nil {
		return entity.Connection{}, err
	}
	return x.Redacted(), nil
}

// SaveConnection validates and upserts. An update that omits the password
// keeps the stored one (the console never sees it).
func (c *Control) SaveConnection(ctx context.Context, conn entity.Connection) (entity.Connection, error) {
	if err := conn.Validate(); err != nil {
		return entity.Connection{}, err
	}
	existing, err := c.conns.Get(ctx, conn.Name)
	switch {
	case err == nil:
		if conn.Password == "" {
			conn.Password = existing.Password
		}
		conn.CreatedAt = existing.CreatedAt
	case errors.Is(err, interfaces.ErrNotFound):
		conn.CreatedAt = time.Now().Unix()
	default:
		return entity.Connection{}, err
	}
	if err := c.conns.Save(ctx, conn); err != nil {
		return entity.Connection{}, err
	}
	return conn.Redacted(), nil
}

// DeleteConnection removes a connection.
func (c *Control) DeleteConnection(ctx context.Context, name string) error {
	return c.conns.Delete(ctx, name)
}

// TestConnection verifies the connection answers.
func (c *Control) TestConnection(ctx context.Context, name string) error {
	conn, err := c.conns.Get(ctx, name)
	if err != nil {
		return err
	}
	return c.db.Ping(ctx, *conn)
}

// connection loads the raw (password-bearing) connection for SQL work.
func (c *Control) connection(ctx context.Context, name string) (*entity.Connection, error) {
	return c.conns.Get(ctx, name)
}

// Op runs one direct SQL operation against a connection. Reads are meant to
// be immediate; writes belong on the change workflow for anything auditable.
func (c *Control) Op(ctx context.Context, name, op, sqlText, table string) (sqlops.Result, error) {
	conn, err := c.connection(ctx, name)
	if err != nil {
		return sqlops.Result{}, err
	}
	switch op {
	case sqlops.OpQuery:
		return c.db.Query(ctx, *conn, sqlText)
	case sqlops.OpAlter:
		return c.db.Alter(ctx, *conn, sqlText)
	case sqlops.OpRollback:
		return c.db.DryRun(ctx, *conn, sqlText)
	case sqlops.OpAnalytics:
		return c.db.Analyze(ctx, *conn, table)
	}
	return sqlops.Result{}, errors.New("unknown op " + op)
}

// Schema introspects a connection for the console's browser.
func (c *Control) Schema(ctx context.Context, name string) ([]sqlops.Table, error) {
	conn, err := c.connection(ctx, name)
	if err != nil {
		return nil, err
	}
	return c.db.Tables(ctx, *conn)
}

// Rows previews a table's rows.
func (c *Control) Rows(ctx context.Context, name, schema, table string, limit int) (sqlops.Result, error) {
	conn, err := c.connection(ctx, name)
	if err != nil {
		return sqlops.Result{}, err
	}
	return c.db.Rows(ctx, *conn, schema, table, limit)
}

// Changes lists every proposed change, newest first.
func (c *Control) Changes(ctx context.Context) ([]entity.Change, error) {
	return c.changes.ListChanges(ctx)
}

// Change returns one change.
func (c *Control) Change(ctx context.Context, id string) (entity.Change, error) {
	ch, err := c.changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// SubmitChange records a change and validates it: both the query and its
// rollback are dry-run inside a transaction that always rolls back, so the
// submitter learns whether it is safe before anyone approves it.
func (c *Control) SubmitChange(ctx context.Context, ch entity.Change) (entity.Change, error) {
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	conn, err := c.connection(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	ch.ID = idgen.UUID()
	ch.CreatedAt = time.Now().Unix()
	ch.Status = constants.ChangeValidated
	ch.Checks = nil
	// The query must be a write (reads run directly).
	kind := entity.ChangeCheck{Target: "query kind"}
	if err := sqlops.VerifyWrite(ch.Query); err != nil {
		kind.Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK = true
		kind.Detail = "write statements only"
	}
	ch.Checks = append(ch.Checks, kind)
	// Then the change and its rollback are dry-run together (nothing persists):
	// the query must apply and the rollback must undo it.
	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := c.db.ValidateChange(ctx, *conn, ch.Query, ch.Rollback)
	switch {
	case err == nil:
		dry.OK = true
		dry.Detail = res.Message
	case errors.Is(err, sqlops.ErrUnverifiable):
		ch.Status = constants.ChangeUnverified
		dry.Error = err.Error()
	default:
		ch.Status = constants.ChangeFailed
		dry.Error = err.Error()
	}
	ch.Checks = append(ch.Checks, dry)
	if err := c.changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// ApproveChange signs off a validated change (self-approval is allowed).
func (c *Control) ApproveChange(ctx context.Context, id string) (entity.Change, error) {
	ch, err := c.changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeValidated && ch.Status != constants.ChangeUnverified {
		return *ch, fmt.Errorf("change is %s, not validated — cannot approve", ch.Status)
	}
	ch.Status = constants.ChangeApproved
	ch.ApprovedAt = time.Now().Unix()
	if err := c.changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// ApplyChange executes an approved change in a transaction and records the
// outcome (a failure is kept on the change, not just returned).
func (c *Control) ApplyChange(ctx context.Context, id string) (entity.Change, error) {
	ch, err := c.changes.GetChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("change is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := c.connection(ctx, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	res, err := c.db.Apply(ctx, *conn, ch.Query)
	if err != nil {
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = c.changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status = constants.ChangeApplied
	ch.AppliedAt = time.Now().Unix()
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := c.changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// RejectChange discards a change.
func (c *Control) RejectChange(ctx context.Context, id string) error {
	return c.changes.DeleteChange(ctx, id)
}
