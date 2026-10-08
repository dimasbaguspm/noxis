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
	"noxis/internal/constants"
	"noxis/internal/entity"
)

const (
	pingTimeout = 10 * time.Second
	stmtTimeout = 30 * time.Second
)

// DB runs verified SQL against registered connections.
type DB struct{}

func New() *DB { return &DB{} }

// Result is the shared operation outcome; SQL fills Columns/Rows for reads
// and EstimatedRows from EXPLAIN when the driver allows it.
type Result = entity.Result

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

// Family is the namespace this module serves.
func (u *DB) Family() string { return constants.FamilySQL }

// Supports reports whether the driver speaks SQL.
func (u *DB) Supports(driver string) bool {
	_, err := sqlDriver(driver)
	return err == nil
}

// Ping verifies a connection answers.
func (u *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
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
	return u.verifyRun(ctx, c, sqlText, reads)
}

// Alter applies writes: DML and DDL; reads are rejected.
func (u *DB) Alter(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	return u.verifyRun(ctx, c, sqlText, writes)
}

func (u *DB) verifyRun(ctx context.Context, c entity.Connection, sqlText string, allowed kindSet) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
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
		started := time.Now()
		var last Result
		n, err := execEach(sqlText, func(stmt string) error {
			var err error
			if reads.set[firstKeyword(stmt)] {
				last, err = query(ctx, db, stmt)
				if err == nil {
					last.EstimatedRows = estimateRows(ctx, db, c.Driver, stmt)
				}
			} else {
				last, err = exec(ctx, db, stmt)
			}
			return hintOf(stmt, err)
		})
		if err != nil {
			return Result{}, err
		}
		if n == 0 {
			return Result{}, fmt.Errorf("no statements to run")
		}
		last.DurationMs = time.Since(started).Milliseconds()
		last.RowsReturned = len(last.Rows)
		last.Message = fmt.Sprintf("%d statement(s) executed", n)
		return last, nil
	})
}

// DryRun runs statements inside a transaction that always rolls back. On
// mysql-family drivers DDL is refused: MySQL commits DDL implicitly, so a
// "dry run" there would apply it.
func (u *DB) DryRun(ctx context.Context, c entity.Connection, sqlText string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if isMySQL(c.Driver) && containsDDL(sqlText) {
		return Result{}, fmt.Errorf("dry run of DDL is not safe on %s (implicit commit) — try it on a postgres connection", c.Driver)
	}
	started := time.Now()
	n, err := inTx(c, func(tx *stdsql.Tx) (int, error) {
		return execEach(sqlText, func(stmt string) error {
			_, err := tx.ExecContext(ctx, stmt)
			return hintOf(stmt, err)
		})
	})
	if err != nil {
		return Result{}, err
	}
	return Result{DurationMs: time.Since(started).Milliseconds(), Message: fmt.Sprintf("rolled back (dry run): %d statement(s)", n)}, nil
}

// Apply executes write statements in a transaction and commits.
//
// ponytail: mysql commits DDL implicitly, so a batch mixing DDL and DML there
// is not all-or-nothing; postgres is.
func (u *DB) Apply(ctx context.Context, c entity.Connection, sqlText string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = verifyKind(sqlText, writes); err != nil {
		return Result{}, err
	}
	started := time.Now()
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
	return Result{
		Affected:   affected,
		DurationMs: time.Since(started).Milliseconds(),
		Message:    fmt.Sprintf("applied: %d statement(s)", n),
	}, nil
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
	ctx, end := logging.StartSpan(ctx)
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
		started := time.Now()
		res, err := query(ctx, db, stmt)
		res.DurationMs = time.Since(started).Milliseconds()
		res.RowsReturned = len(res.Rows)
		return res, err
	})
}

// Tables lists user relations and their columns.
func (u *DB) Tables(ctx context.Context, c entity.Connection) (tables []Table, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return withDB(c, func(db *stdsql.DB) ([]Table, error) {
		return listTables(ctx, db)
	})
}

// Analyze refreshes planner statistics (ANALYZE / ANALYZE TABLE).
func (u *DB) Analyze(ctx context.Context, c entity.Connection, table string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
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

// Stats snapshots the instance (connections, size, cache, throughput).
func (u *DB) Stats(ctx context.Context, c entity.Connection) (stats Stats, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return withDB(c, func(db *stdsql.DB) (Stats, error) {
		ctx, cancel := context.WithTimeout(ctx, stmtTimeout)
		defer cancel()
		return collectStats(ctx, db, c)
	})
}

// VerifyWrite exposes the write guard (a change's query must be a write).
func (u *DB) VerifyWrite(sqlText string) error { return verifyKind(sqlText, writes) }

// Kind classifies a batch as dml, ddl, or read (mixed batches report ddl when
// any DDL appears — that is the stricter review).
func (u *DB) Kind(sqlText string) string {
	kind := constants.KindDML
	for _, stmt := range splitStatements(sqlText) {
		k := firstKeyword(stmt)
		if k == "" {
			continue
		}
		if ddls.set[k] {
			return constants.KindDDL
		}
		if !reads.set[k] && !writes.set[k] {
			continue
		}
	}
	return kind
}
