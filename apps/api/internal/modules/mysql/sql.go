// Package mysql runs verified SQL over mysql connections: reads, writes, dry-run transactions, analyze, and schema introspection.
package mysql

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const (
	pingTimeout = 10 * time.Second
	stmtTimeout = 30 * time.Second
)

// DB runs verified SQL against this driver's connections.
type DB struct{ info infoCache }

// New builds the module; it serves one driver and refuses the rest.
func New() *DB { return &DB{} }

// Result is the shared operation outcome; SQL fills Columns/Rows for reads and EstimatedRows from EXPLAIN when the driver allows it.
type Result = entity.Result

// Satisfiers: the SQL engine serves every capability the console asks a SQL connection for.
var (
	_ interfaces.ConsoleQuerier  = (*DB)(nil)
	_ interfaces.SchemaReader    = (*DB)(nil)
	_ interfaces.ServerReporter  = (*DB)(nil)
	_ interfaces.ChangeProver    = (*DB)(nil)
	_ interfaces.ChangeApplier   = (*DB)(nil)
	_ interfaces.SQLChangeEngine = (*DB)(nil)
)

// withDB opens a short-lived pool (control-plane rates) for one operation.
func withDB[T any](c entity.Connection, fn func(db *stdsql.DB) (T, error)) (T, error) {
	var zero T
	d, err := dialectOf(c.Driver)
	if err != nil {
		return zero, err
	}
	conn, err := dsn(c)
	if err != nil {
		return zero, err
	}
	db, err := stdsql.Open(d.driver, conn)
	if err != nil {
		return zero, fmt.Errorf("open %s: %w", c.Name, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(d.maxOpenConns)
	db.SetMaxIdleConns(d.maxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	return fn(db)
}

// execEach runs every non-empty statement of a batch through fn, numbering failures and returning how many ran.
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

// Driver is the driver this module serves.
func (db *DB) Driver() string { return constants.DriverMySQL }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(driver string) bool { return strings.EqualFold(driver, constants.DriverMySQL) }

// Ping verifies a connection answers.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
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

// Query runs read-only statements; DML and DDL are rejected (writes go through the change workflow).
func (db *DB) Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	return db.verifyRun(ctx, c, sqlText, reads)
}

func (db *DB) verifyRun(ctx context.Context, c entity.Connection, sqlText string, allowed kindSet) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = verifyKind(sqlText, allowed); err != nil {
		return Result{}, err
	}
	return db.run(ctx, c, sqlText)
}

// run executes each statement separately (verified incrementally, no multi-statement driver quirks) and returns the last statement's outcome.
func (db *DB) run(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
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
			return Result{}, errNoStatements
		}
		last.DurationMs = time.Since(started).Milliseconds()
		last.RowsReturned = len(last.Rows)
		last.Message = fmt.Sprintf("%d statement(s) executed", n)
		return last, nil
	})
}

// DryRun always rolls back; it refuses DDL whose server commits implicitly, which would make the dry run destructive.
func (db *DB) DryRun(ctx context.Context, c entity.Connection, sqlText string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	d, derr := dialectOf(c.Driver)
	if derr != nil {
		return Result{}, derr
	}
	// an engine that commits DDL implicitly would apply the batch instead of rolling it back
	if d.txn == txnWithoutDDL && containsDDL(sqlText) {
		return Result{}, fmt.Errorf("dry run of DDL is not safe on %s (implicit commit) — try it on a postgres connection", c.Driver)
	}
	started := time.Now()
	_, n, err := runBatch(ctx, c, sqlText, true)
	if err != nil {
		return Result{}, err
	}
	if n == 0 {
		return Result{}, errNoStatements
	}
	return Result{DurationMs: time.Since(started).Milliseconds(), Message: fmt.Sprintf("rolled back (dry run): %d statement(s)", n)}, nil
}

// Apply executes write statements in a transaction and commits.
//
// shortcut: MySQL commits DDL implicitly, so mixed DDL/DML batches are not atomic; split them or use PostgreSQL when atomicity is required.
func (db *DB) Apply(ctx context.Context, c entity.Connection, sqlText string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err = verifyKind(sqlText, writes); err != nil {
		return Result{}, err
	}
	started := time.Now()
	affected, n, err := runBatch(ctx, c, sqlText, false)
	if err != nil {
		return Result{}, err
	}
	if n == 0 {
		return Result{}, errNoStatements
	}
	return Result{
		Affected:   affected,
		DurationMs: time.Since(started).Milliseconds(),
		Message:    fmt.Sprintf("applied: %d statement(s)", n),
	}, nil
}

// errNoStatements reports an empty batch: a change or a console op with nothing to run is a caller mistake, not a success.
var errNoStatements = errors.New("no statements to run")

// runBatch executes every statement of a batch inside one transaction and commits unless rollback is set.
func runBatch(ctx context.Context, c entity.Connection, sqlText string, rollback bool) (affected int64, n int, err error) {
	n, err = inTx(c, rollback, func(tx *stdsql.Tx) (int, error) {
		return execEach(sqlText, func(stmt string) error {
			out, err := tx.ExecContext(ctx, stmt)
			if err != nil {
				return hintOf(stmt, err)
			}
			if a, err := out.RowsAffected(); err == nil {
				affected += a
			}
			return nil
		})
	})
	return affected, n, err
}

// inTx commits on success unless rollback is set; dry runs leave data untouched.
func inTx(c entity.Connection, rollback bool, fn func(tx *stdsql.Tx) (int, error)) (int, error) {
	return withDB(c, func(db *stdsql.DB) (int, error) {
		tx, err := db.Begin()
		if err != nil {
			return 0, fmt.Errorf("begin: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		n, err := fn(tx)
		if err != nil || rollback {
			return n, err
		}
		if err := tx.Commit(); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return n, nil
	})
}

// Rows previews a table's rows (bounded).
func (db *DB) Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (res Result, err error) {
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
func (db *DB) Tables(ctx context.Context, c entity.Connection) (tables []entity.Table, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return withDB(c, func(db *stdsql.DB) ([]entity.Table, error) {
		return listTables(ctx, db, c.Driver)
	})
}

// Analyze refreshes planner statistics (ANALYZE / ANALYZE TABLE).
func (db *DB) Analyze(ctx context.Context, c entity.Connection, table string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if !identOK(table) {
		return Result{}, fmt.Errorf("invalid table name %q", table)
	}
	d, err := dialectOf(c.Driver)
	if err != nil {
		return Result{}, err
	}
	stmt := d.analyzeStmt(table)
	return db.run(ctx, c, stmt)
}

// Referenced reports the tables and columns a batch mentions, so the change record can list what it touches.
func (db *DB) Referenced(sqlText string) (tables []string, columns []string) {
	return Referenced(sqlText)
}

// VerifyWrite exposes the write guard (a change's query must be a write).
func (db *DB) VerifyWrite(sqlText string) error { return verifyKind(sqlText, writes) }

// Kind classifies a batch as dml, ddl, or read (mixed batches report ddl when any DDL appears — that is the stricter review).
func (db *DB) Kind(sqlText string) string {
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
