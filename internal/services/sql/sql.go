// Package sql runs verified SQL against registered connections: reads,
// writes, dry-run transactions, analyze, and schema introspection. Statement
// kinds are checked before execute, per statement.
package sql

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

const (
	pingTimeout = 10 * time.Second
	stmtTimeout = 30 * time.Second
)

// DB runs verified SQL against registered connections.
type DB struct{}

func New() *DB { return &DB{} }

// Result is one operation's outcome: a grid for reads, counts for writes.
type Result struct {
	Columns  []string   `json:"columns,omitempty"`
	Rows     [][]string `json:"rows,omitempty"`
	Affected int64      `json:"affected"`
	Message  string     `json:"message"`
}

// Table is a browsable relation; Column one of its attributes.
type Table struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// span opens a span for one SQL operation.
func span(ctx context.Context, name string) (ctxOut context.Context, end func(error)) {
	ctxOut, end = logging.StartSpan(ctx, "sql."+name)
	return ctxOut, end
}

// withDB opens a short-lived pool (control-plane rates) for one operation.
func withDB[T any](c entity.Connection, fn func(db *stdsql.DB) (T, error)) (T, error) {
	var zero T
	driver, err := sqlDriver(c.Driver)
	if err != nil {
		return zero, err
	}
	conn, err := dsn(c)
	if err != nil {
		return zero, err
	}
	db, err := stdsql.Open(driver, conn)
	if err != nil {
		return zero, fmt.Errorf("open %s: %w", c.Name, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return fn(db)
}

// execEach runs every non-empty statement of a batch through fn, numbering
// failures and returning how many ran.
func execEach(sqlText string, fn func(stmt string) error) (int, error) {
	n := 0
	for _, stmt := range splitStatements(sqlText) {
		if firstKeyword(stmt) == "" {
			continue
		}
		n++
		if err := fn(stmt); err != nil {
			return n, fmt.Errorf("statement %d: %w", n, err)
		}
	}
	return n, nil
}

// Ping verifies a connection answers.
func (u *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := span(ctx, "ping")
	defer func() { end(err) }()
	_, err = withDB(c, func(db *stdsql.DB) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, pingTimeout)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return struct{}{}, fmt.Errorf("ping %s: %w", c.Name, err)
		}
		return struct{}{}, nil
	})
	return err
}

// Query runs read-only statements; DML and DDL are rejected (writes belong on
// Alter or the change workflow).
func (u *DB) Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	return u.verifyRun(ctx, c, sqlText, reads, "query")
}

// Alter applies writes: DML and DDL; reads are rejected.
func (u *DB) Alter(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	return u.verifyRun(ctx, c, sqlText, writes, "alter")
}

func (u *DB) verifyRun(ctx context.Context, c entity.Connection, sqlText string, allowed kindSet, name string) (res Result, err error) {
	ctx, end := span(ctx, name)
	defer func() { end(err) }()
	if err = verifyKind(sqlText, allowed); err != nil {
		return Result{}, err
	}
	return u.run(ctx, c, sqlText)
}

// run executes each statement separately (verified incrementally, no
// multi-statement driver quirks) and returns the last statement's outcome.
func (u *DB) run(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	return withDB(c, func(db *stdsql.DB) (Result, error) {
		ctx, cancel := context.WithTimeout(ctx, stmtTimeout)
		defer cancel()
		var last Result
		n, err := execEach(sqlText, func(stmt string) error {
			var err error
			if reads.set[firstKeyword(stmt)] {
				last, err = query(ctx, db, stmt)
			} else {
				last, err = exec(ctx, db, stmt)
			}
			return err
		})
		if err != nil {
			return Result{}, err
		}
		if n == 0 {
			return Result{}, fmt.Errorf("no statements to run")
		}
		last.Message = fmt.Sprintf("%d statement(s) executed", n)
		return last, nil
	})
}

// DryRun runs statements inside a transaction that always rolls back. On
// mysql-family drivers DDL is refused: MySQL commits DDL implicitly, so a
// "dry run" there would apply it.
func (u *DB) DryRun(ctx context.Context, c entity.Connection, sqlText string) (res Result, err error) {
	ctx, end := span(ctx, "rollback")
	defer func() { end(err) }()
	if isMySQL(c.Driver) && containsDDL(sqlText) {
		return Result{}, fmt.Errorf("dry run of DDL is not safe on %s (implicit commit) — try it on a postgres connection", c.Driver)
	}
	n, err := inTx(c, func(tx *stdsql.Tx) (int, error) {
		return execEach(sqlText, func(stmt string) error {
			_, err := tx.ExecContext(ctx, stmt)
			return err
		})
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Message: fmt.Sprintf("rolled back (dry run): %d statement(s)", n)}, nil
}

// ValidateChange dry-runs a change and its rollback together: the query must
// apply cleanly, then the rollback must undo it, all inside one transaction
// that always rolls back. This is why a rollback is validated against the
// change rather than against the current database.
func (u *DB) ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (res Result, err error) {
	ctx, end := span(ctx, "validate")
	defer func() { end(err) }()
	if isMySQL(c.Driver) && containsDDL(query) {
		return Result{}, fmt.Errorf("query has DDL and %s: %w", c.Driver, ErrUnverifiable)
	}
	n, err := inTx(c, func(tx *stdsql.Tx) (int, error) {
		if _, err := execEach(query, func(stmt string) error {
			_, err := tx.ExecContext(ctx, stmt)
			return err
		}); err != nil {
			return 0, fmt.Errorf("query %w", err)
		}
		return execEach(rollback, func(stmt string) error {
			_, err := tx.ExecContext(ctx, stmt)
			return err
		})
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Message: fmt.Sprintf("query and rollback both valid (rolled back, %d rollback statement(s))", n)}, nil
}

// Apply executes write statements in a transaction and commits.
//
// ponytail: mysql commits DDL implicitly, so a batch mixing DDL and DML there
// is not all-or-nothing; postgres is.
func (u *DB) Apply(ctx context.Context, c entity.Connection, sqlText string) (res Result, err error) {
	ctx, end := span(ctx, "apply")
	defer func() { end(err) }()
	if err = verifyKind(sqlText, writes); err != nil {
		return Result{}, err
	}
	var affected int64
	n, err := inTx(c, func(tx *stdsql.Tx) (int, error) {
		return execEach(sqlText, func(stmt string) error {
			out, err := tx.ExecContext(ctx, stmt)
			if err != nil {
				return err
			}
			if a, err := out.RowsAffected(); err == nil {
				affected += a
			}
			return nil
		})
	})
	if err != nil {
		return Result{}, err
	}
	if n == 0 {
		return Result{}, fmt.Errorf("no statements to apply")
	}
	return Result{Affected: affected, Message: fmt.Sprintf("applied: %d statement(s)", n)}, nil
}

// inTx runs fn inside a transaction: commit when it returns nil error.
func inTx(c entity.Connection, fn func(tx *stdsql.Tx) (int, error)) (int, error) {
	return withDB(c, func(db *stdsql.DB) (int, error) {
		tx, err := db.Begin()
		if err != nil {
			return 0, fmt.Errorf("begin: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		n, err := fn(tx)
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return n, nil
	})
}

// Rows previews a table's rows (bounded).
func (u *DB) Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (res Result, err error) {
	ctx, end := span(ctx, "rows")
	defer func() { end(err) }()
	if !identOK(table) || (schema != "" && !identOK(schema)) {
		return Result{}, fmt.Errorf("invalid table %q", table)
	}
	if limit <= 0 || limit > maxPreviewRows {
		limit = defaultPreviewRows
	}
	stmt := "SELECT * FROM " + qualify(c, schema, table) + " LIMIT " + itoa(limit)
	return withDB(c, func(db *stdsql.DB) (Result, error) {
		ctx, cancel := context.WithTimeout(ctx, stmtTimeout)
		defer cancel()
		return query(ctx, db, stmt)
	})
}

// Tables lists user relations and their columns.
func (u *DB) Tables(ctx context.Context, c entity.Connection) (tables []Table, err error) {
	ctx, end := span(ctx, "schema")
	defer func() { end(err) }()
	return withDB(c, func(db *stdsql.DB) ([]Table, error) {
		return listTables(ctx, db)
	})
}

// Analyze refreshes planner statistics (ANALYZE / ANALYZE TABLE).
func (u *DB) Analyze(ctx context.Context, c entity.Connection, table string) (res Result, err error) {
	ctx, end := span(ctx, "analyze")
	defer func() { end(err) }()
	if !identOK(table) {
		return Result{}, fmt.Errorf("invalid table name %q", table)
	}
	stmt := "ANALYZE " + table
	if isMySQL(c.Driver) {
		stmt = "ANALYZE TABLE " + table
	}
	return u.run(ctx, c, stmt)
}

// VerifyWrite exposes the write guard (a change's query must be a write).
func (u *DB) VerifyWrite(sqlText string) error { return verifyKind(sqlText, writes) }
