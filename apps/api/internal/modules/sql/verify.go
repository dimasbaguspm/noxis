package sql

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"strings"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// scratchPrefix names the throwaway schema each change is verified in. The
// random suffix keeps concurrent submissions from colliding.
const scratchPrefix = "noxis_verify_"

// ValidateChange verifies a change and its rollback in a throwaway scratch
// schema on the target server: the objects the batch references are mirrored
// there structure-only (no rows), the query and its rollback run against the
// scratch, and the scratch is dropped on the way out — success or failure.
//
// This is what makes MySQL/MariaDB DDL verifiable: mysql commits DDL
// implicitly, so a transaction dry run was never safe there; a scratch schema
// is.
//
// ponytail: the batch resolves unqualified names against the scratch (pg
// search_path, mysql USE); a statement that names a live schema explicitly is
// the operator's own call. Add SQL rewriting if that shows up.
func (u *DB) ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	// SQLite DDL is transactional, so a rolled-back transaction is a safe dry
	// run on the live schema; no scratch database is needed.
	if isSQLite(c.Driver) {
		return validateChangeTx(ctx, c, query, rollback)
	}
	started := time.Now()
	schema := scratchPrefix + idgen.Hex(6)

	var mirrored []string
	var affected int64
	_, err = withDB(c, func(db *stdsql.DB) (struct{}, error) {
		conn, err := db.Conn(ctx)
		if err != nil {
			return struct{}{}, fmt.Errorf("scratch connection: %w", err)
		}
		defer func() { _ = conn.Close() }()

		if err := createScratch(ctx, conn, c.Driver, schema); err != nil {
			return struct{}{}, err
		}
		// Cleanup must survive a cancelled request: the schema exists on the
		// server the moment it is created.
		defer func() { _ = dropScratch(context.WithoutCancel(ctx), conn, c.Driver, schema) }()

		mirrored, err = mirrorReferenced(ctx, conn, c, schema, query+"\n"+rollback)
		if err != nil {
			return struct{}{}, err
		}
		if err := useScratch(ctx, conn, c.Driver, schema); err != nil {
			return struct{}{}, err
		}
		if _, err := execEach(query, func(stmt string) error {
			out, err := conn.ExecContext(ctx, stmt)
			if err != nil {
				return hintOf(stmt, err)
			}
			if a, err := out.RowsAffected(); err == nil {
				affected += a
			}
			return nil
		}); err != nil {
			return struct{}{}, fmt.Errorf("query %w", err)
		}
		if _, err := execEach(rollback, func(stmt string) error {
			_, err := conn.ExecContext(ctx, stmt)
			return hintOf(stmt, err)
		}); err != nil {
			return struct{}{}, fmt.Errorf("rollback %w", err)
		}
		return struct{}{}, nil
	})
	if err != nil {
		return Result{}, err
	}

	detail := fmt.Sprintf("verified in scratch schema %q on %s", schema, strings.ToLower(c.Driver))
	if len(mirrored) > 0 {
		detail += "; mirrored " + strings.Join(mirrored, ", ")
	}
	return Result{
		Affected:      affected,
		DurationMs:    time.Since(started).Milliseconds(),
		Message:       detail,
		ScratchSchema: schema,
	}, nil
}

// validateChangeTx dry-runs a change and its rollback in one transaction that
// always rolls back. SQLite is the driver this exists for: it supports
// transactional DDL, so the live tables can be exercised directly.
func validateChangeTx(ctx context.Context, c entity.Connection, query, rollback string) (Result, error) {
	started := time.Now()
	var affected int64
	_, err := withDB(c, func(db *stdsql.DB) (struct{}, error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return struct{}{}, fmt.Errorf("begin: %w", err)
		}
		defer func() { _ = tx.Rollback() }() // the dry run never persists
		if _, err := execEach(query, func(stmt string) error {
			out, err := tx.ExecContext(ctx, stmt)
			if err != nil {
				return hintOf(stmt, err)
			}
			if a, err := out.RowsAffected(); err == nil {
				affected += a
			}
			return nil
		}); err != nil {
			return struct{}{}, fmt.Errorf("query %w", err)
		}
		if _, err := execEach(rollback, func(stmt string) error {
			_, err := tx.ExecContext(ctx, stmt)
			return hintOf(stmt, err)
		}); err != nil {
			return struct{}{}, fmt.Errorf("rollback %w", err)
		}
		return struct{}{}, nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		Affected:   affected,
		DurationMs: time.Since(started).Milliseconds(),
		Message:    "verified in a rolled-back transaction on sqlite",
	}, nil
}

// mirrorReferenced copies the structure of every referenced table that already
// exists live into the scratch schema, so the change has something to act on.
// Tables the batch creates itself are skipped.
func mirrorReferenced(ctx context.Context, conn *stdsql.Conn, c entity.Connection, scratch, sqlText string) ([]string, error) {
	created := createdTables(sqlText)
	refs, _ := Referenced(sqlText)
	var mirrored []string
	for _, ref := range refs {
		if created[ref] {
			continue
		}
		schema, table := splitRef(c, ref)
		if !identOK(schema) || !identOK(table) || created[schema+"."+table] {
			continue
		}
		exists, err := tableExists(ctx, conn, schema, table)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		if err := mirrorTable(ctx, conn, c.Driver, scratch, schema, table); err != nil {
			return nil, fmt.Errorf("mirror %s.%s: %w", schema, table, err)
		}
		mirrored = append(mirrored, schema+"."+table)
	}
	return mirrored, nil
}

// splitRef resolves a referenced table to its schema; a bare name lives in
// the connection's default schema, the same place the live statement would
// have resolved it.
func splitRef(c entity.Connection, ref string) (schema, table string) {
	if i := strings.LastIndex(ref, "."); i > 0 {
		return ref[:i], ref[i+1:]
	}
	if isMySQL(c.Driver) {
		return c.Database, ref
	}
	if isSQLite(c.Driver) {
		return schemaSQLite, ref
	}
	return schemaPostgres, ref
}

// tableExists reports whether a base table exists in the live schema.
func tableExists(ctx context.Context, conn *stdsql.Conn, schema, table string) (bool, error) {
	// identifiers were validated by identOK; placeholders are not portable
	// between pgx and the mysql driver.
	stmt := fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s' AND table_type = 'BASE TABLE'",
		schema, table)
	var n int
	if err := conn.QueryRowContext(ctx, stmt).Scan(&n); err != nil {
		return false, fmt.Errorf("look up %s.%s: %w", schema, table, err)
	}
	return n > 0, nil
}

// mirrorTable creates a structure-only copy (columns, defaults, indexes,
// constraints) of a live table in the scratch schema. No rows are copied.
func mirrorTable(ctx context.Context, conn *stdsql.Conn, driver, scratch, schema, table string) error {
	target, source := quoteRef(driver, scratch, table), quoteRef(driver, schema, table)
	stmt := "CREATE TABLE " + target + " (LIKE " + source + " INCLUDING ALL)"
	if isMySQL(driver) {
		stmt = "CREATE TABLE " + target + " LIKE " + source
	}
	_, err := conn.ExecContext(ctx, stmt)
	return err
}

func createScratch(ctx context.Context, conn *stdsql.Conn, driver, schema string) error {
	stmt := `CREATE SCHEMA "` + schema + `"`
	if isMySQL(driver) {
		stmt = "CREATE DATABASE `" + schema + "`"
	}
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create scratch schema: %w", err)
	}
	return nil
}

func dropScratch(ctx context.Context, conn *stdsql.Conn, driver, schema string) error {
	stmt := `DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`
	if isMySQL(driver) {
		stmt = "DROP DATABASE IF EXISTS `" + schema + "`"
	}
	_, err := conn.ExecContext(ctx, stmt)
	return err
}

// useScratch points the pinned connection at the scratch schema so unqualified
// names in the change resolve there.
func useScratch(ctx context.Context, conn *stdsql.Conn, driver, schema string) error {
	stmt := `SET search_path TO "` + schema + `", public`
	if isMySQL(driver) {
		stmt = "USE `" + schema + "`"
	}
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("select scratch schema: %w", err)
	}
	return nil
}
