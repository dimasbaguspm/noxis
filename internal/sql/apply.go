package sql

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"noxis/internal/entity"
)

// Apply executes write statements inside a transaction and commits. Only the
// approved change path calls this — direct writes go through Alter.
//
// ponytail: mysql commits DDL implicitly, so a multi-statement batch that
// mixes DDL and DML there is not all-or-nothing; postgres is.
func (u *DB) Apply(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	if err := verifyKind(sqlText, writes); err != nil {
		return Result{}, err
	}
	db, err := open(c)
	if err != nil {
		return Result{}, err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var affected int64
	n := 0
	for _, stmt := range splitStatements(sqlText) {
		if firstKeyword(stmt) == "" {
			continue
		}
		out, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return Result{}, fmt.Errorf("statement %d: %w", n+1, err)
		}
		n++
		if a, err := out.RowsAffected(); err == nil {
			affected += a
		}
	}
	if n == 0 {
		return Result{}, fmt.Errorf("no statements to apply")
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("commit: %w", err)
	}
	return Result{Affected: affected, Message: fmt.Sprintf("applied: %d statement(s)", n)}, nil
}

// ErrUnverifiable reports that a change cannot be dry-run on this driver
// (mysql commits DDL implicitly), so validation must be skipped honestly.
var ErrUnverifiable = errors.New("unverifiable on this driver")

// ValidateChange dry-runs a change and its rollback together, in one
// transaction that always rolls back: the query must apply cleanly and the
// rollback must then undo it. Nothing persists — this is the "hidden schema"
// confidence check, and it is why a rollback is validated against the change
// rather than against the current database.
func (u *DB) ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (Result, error) {
	if isMySQL(c.Driver) && containsDDL(query) {
		return Result{}, fmt.Errorf("query has DDL and %s: %w", c.Driver, ErrUnverifiable)
	}
	db, err := open(c)
	if err != nil {
		return Result{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for i, stmt := range splitStatements(query) {
		if firstKeyword(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return Result{}, fmt.Errorf("query statement %d: %w", i+1, err)
		}
	}
	n := 0
	for i, stmt := range splitStatements(rollback) {
		if firstKeyword(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return Result{}, fmt.Errorf("rollback statement %d: %w", i+1, err)
		}
		n++
	}
	return Result{Message: fmt.Sprintf("query and rollback both valid (rolled back, %d rollback statement(s))", n)}, nil
}

// containsDDL reports whether any statement is a DDL kind.
func containsDDL(sqlText string) bool {
	for _, stmt := range splitStatements(sqlText) {
		if ddls.set[firstKeyword(stmt)] {
			return true
		}
	}
	return false
}

// Rows returns a table's rows (bounded preview for the console browser).
func (u *DB) Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (Result, error) {
	if !identOK(table) || (schema != "" && !identOK(schema)) {
		return Result{}, fmt.Errorf("invalid table %q", table)
	}
	if limit <= 0 || limit > maxPreviewRows {
		limit = defaultPreviewRows
	}
	db, err := open(c)
	if err != nil {
		return Result{}, err
	}
	defer db.Close()
	stmt := "SELECT * FROM " + qualify(c, schema, table) + " LIMIT " + strconv.Itoa(limit)
	return query(ctx, db, stmt)
}

// qualify quotes a table reference per driver (identifiers were validated).
func qualify(c entity.Connection, schema, table string) string {
	if isMySQL(c.Driver) {
		if schema != "" {
			return "`" + schema + "`.`" + table + "`"
		}
		return "`" + table + "`"
	}
	if schema != "" {
		return `"` + schema + `"."` + table + `"`
	}
	return `"` + table + `"`
}
