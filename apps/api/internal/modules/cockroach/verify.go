package cockroach

import (
	"context"
	"crypto/sha256"
	stdsql "database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"noxis/internal/common/logging"
	"noxis/internal/common/sqltext"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// verifySampleRows bounds how much of a referenced table is copied and hashed during validation.
const verifySampleRows = 100

// qualifiedIdent matches an identifier with a dot-qualified part, including column references that a scratch search_path would not rewrite.
var qualifiedIdent = regexp.MustCompile("(?i)(?:[a-z_][a-z0-9_$]*|\"[^\"]+\"|`[^`]+`)\\s*\\.\\s*(?:[a-z_*][a-z0-9_$]*|\"[^\"]+\"|`[^`]+`)")

// ValidateChange proves a batch by rolling it back in the live transaction: this engine has transactional DDL, so a
// throwaway schema would prove less than the real thing.
func (db *DB) ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	return validateChangeTx(ctx, c, query, rollback)
}

// hasQualifiedIdent finds schema.table and table.column references; either can escape a scratch search_path.
func hasQualifiedIdent(text string) bool {
	for _, stmt := range sqltext.SplitStatements(text) {
		if qualifiedIdent.MatchString(stmt) {
			return true
		}
	}
	return false
}

// verifyRef is one table a change can read or write, resolved to the schema the validation runs in.
type verifyRef struct {
	schema string
	table  string
}

func (r verifyRef) key() string { return r.schema + "." + r.table }

// verificationRefs resolves every table named by the change, including tables the query creates.
func verificationRefs(c entity.Connection, sqlText, scratch string) []verifyRef {
	created := sqltext.CreatedTables(sqlText)
	names, _ := sqltext.Referenced(sqlText)
	for name := range created {
		names = append(names, name)
	}
	seen := map[string]bool{}
	var refs []verifyRef
	for _, name := range names {
		schema, table := splitRef(c, name)
		// Unqualified names in scratch resolve to the scratch schema, not the live default.
		if scratch != "" && !strings.Contains(name, ".") {
			schema = scratch
		}
		if !sqltext.IdentOK(schema) || !sqltext.IdentOK(table) {
			continue
		}
		key := schema + "." + table
		if seen[key] {
			continue
		}
		seen[key] = true
		refs = append(refs, verifyRef{schema: schema, table: table})
	}
	return refs
}

// tableState is the part of a table that must be identical before and after a rollback.
type tableState struct {
	exists bool
	rows   int64
	sample int
	digest string
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*stdsql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *stdsql.Row
}

// captureStates fingerprints every referenced table, including absence, so a created or dropped table is compared too.
func captureStates(ctx context.Context, q queryer, driver string, refs []verifyRef) (map[string]tableState, error) {
	states := make(map[string]tableState, len(refs))
	for _, ref := range refs {
		exists, err := tablePresent(ctx, q, driver, ref.schema, ref.table)
		if err != nil {
			return nil, err
		}
		if !exists {
			states[ref.key()] = tableState{}
			continue
		}
		state, err := snapshotTable(ctx, q, driver, ref.schema, ref.table)
		if err != nil {
			return nil, err
		}
		states[ref.key()] = state
	}
	return states, nil
}

// tablePresent asks the catalog, not the data, so a missing table is a normal state rather than a query error.
func tablePresent(ctx context.Context, q queryer, driver, schema, table string) (bool, error) {
	d, err := dialectOf(driver)
	if err != nil {
		return false, err
	}
	// identifiers are validated by identOK; the statement inlines them because the engines do not share placeholder syntax.
	var n int
	if err := q.QueryRowContext(ctx, d.tablePresentStatement(schema, table)).Scan(&n); err != nil {
		return false, fmt.Errorf("look up %s.%s: %w", schema, table, err)
	}
	return n > 0, nil
}

// snapshotTable records the full row count and a deterministic hash of a bounded row sample.
func snapshotTable(ctx context.Context, q queryer, driver, schema, table string) (tableState, error) {
	ref := quoteRef(driver, schema, table)
	var count int64
	if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+ref).Scan(&count); err != nil {
		return tableState{}, fmt.Errorf("count %s.%s: %w", schema, table, err)
	}
	rows, err := q.QueryContext(ctx, "SELECT * FROM "+ref+" LIMIT "+itoa(verifySampleRows))
	if err != nil {
		return tableState{}, fmt.Errorf("sample %s.%s: %w", schema, table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return tableState{}, err
	}
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	var sample []string
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return tableState{}, fmt.Errorf("sample %s.%s: %w", schema, table, err)
		}
		sample = append(sample, encodeRow(values))
	}
	if err := rows.Err(); err != nil {
		return tableState{}, fmt.Errorf("sample %s.%s: %w", schema, table, err)
	}
	// The database may return the sample in any order; sorting makes the hash order-independent.
	sort.Strings(sample)
	hash := sha256.New()
	for _, row := range sample {
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(row)))
		_, _ = hash.Write(size[:])
		_, _ = hash.Write([]byte(row))
	}
	return tableState{
		exists: true,
		rows:   count,
		sample: len(sample),
		digest: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func encodeRow(values []any) string {
	var b strings.Builder
	for _, value := range values {
		var text string
		switch v := value.(type) {
		case nil:
			text = "\x00"
		case []byte:
			text = string(v)
		default:
			text = fmt.Sprint(v)
		}
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(text)))
		b.Write(size[:])
		b.WriteString(text)
	}
	return b.String()
}

// compareStates returns the first table whose row count, sample or digest changed.
func compareStates(before, after map[string]tableState, refs []verifyRef) error {
	for _, ref := range refs {
		key := ref.key()
		a, b := before[key], after[key]
		if a.exists != b.exists || a.rows != b.rows || a.sample != b.sample || a.digest != b.digest {
			if !a.exists {
				return fmt.Errorf("rollback left table %s behind", key)
			}
			if !b.exists {
				return fmt.Errorf("rollback removed table %s", key)
			}
			return fmt.Errorf("rollback did not restore %s (rows %d -> %d)", key, a.rows, b.rows)
		}
	}
	return nil
}

func sampledRows(states map[string]tableState) int {
	total := 0
	for _, state := range states {
		if state.exists {
			total += state.sample
		}
	}
	return total
}

func changeSummary(refs []verifyRef, states map[string]tableState) string {
	return fmt.Sprintf("checked %d table(s), %d sampled row(s)", len(refs), sampledRows(states))
}

// retryableCode is the SQLSTATE a serializable engine returns when a transaction loses a race: the statement is fine
// and the transaction must be replayed, so it is a retry and never a failed change.
const retryableCode = "40001"

// retryBudget bounds how many times a contended dry run is replayed before the engine's own reason is reported.
const retryBudget = 3

// retryable reports the lost-race code, so no other engine error is replayed.
func retryable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == retryableCode
}

// replay retries a dry run while the engine reports a lost race.
func replay(run func() (Result, error)) (Result, error) {
	res, err := run()
	for attempt := 1; attempt < retryBudget && retryable(err); attempt++ {
		res, err = run()
	}
	return res, err
}

// ValidateChange proves a batch by rolling it back in the live transaction, replaying it when cockroach reports that
// another transaction won the race.
func validateChangeTx(ctx context.Context, c entity.Connection, query, rollback string) (Result, error) {
	return replay(func() (Result, error) { return dryRunTx(ctx, c, query, rollback) })
}

// dryRunTx exercises live tables because the whole transaction is rolled back.
func dryRunTx(ctx context.Context, c entity.Connection, query, rollback string) (Result, error) {
	started := time.Now()
	var affected int64
	var summary string
	refs := verificationRefs(c, query+"\n"+rollback, "")
	_, err := withDB(c, func(db *stdsql.DB) (struct{}, error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return struct{}{}, fmt.Errorf("begin: %w", err)
		}
		defer func() { _ = tx.Rollback() }() // the dry run never persists
		before, err := captureStates(ctx, tx, c.Driver, refs)
		if err != nil {
			return struct{}{}, err
		}
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
		after, err := captureStates(ctx, tx, c.Driver, refs)
		if err != nil {
			return struct{}{}, err
		}
		if err := compareStates(before, after, refs); err != nil {
			return struct{}{}, err
		}
		if !sqltext.ContainsDDL(query, ddls.set) && sampledRows(before) == 0 {
			return struct{}{}, interfaces.Unverifiable(constants.UnverifiedNoRows, "no sampled rows for the tables this change touches")
		}
		summary = changeSummary(refs, before)
		return struct{}{}, nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		Affected:   affected,
		DurationMs: time.Since(started).Milliseconds(),
		Message:    "verified in a rolled-back transaction; " + summary,
	}, nil
}

// splitRef resolves a referenced table to its schema; a bare name lives in the connection's default schema, the same place the live statement would have resolved it.
func splitRef(c entity.Connection, ref string) (schema, table string) {
	if i := strings.LastIndex(ref, "."); i > 0 {
		return ref[:i], ref[i+1:]
	}
	d, err := dialectOf(c.Driver)
	if err != nil || d.schema == "" {
		// a bare name in a database-per-schema engine resolves to the connection's own database
		return c.Database, ref
	}
	return d.schema, ref
}
