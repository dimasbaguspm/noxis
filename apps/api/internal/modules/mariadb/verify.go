package mariadb

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

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/common/sqltext"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// scratchPrefix names the throwaway schema each change is verified in.
const scratchPrefix = "noxis_verify_"

// verifySampleRows bounds how much of a referenced table is copied and hashed during validation.
const verifySampleRows = 100

// ValidateChange runs query and rollback in the proof this profile chose: a rolled-back transaction where DDL is transactional, a scratch schema where it is not.
func (db *DB) ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	d, derr := dialectOf(c.Driver)
	if derr != nil {
		return Result{}, derr
	}
	text := query + "\n" + rollback
	if d.transactional(text) {
		return validateChangeTx(ctx, c, query, rollback)
	}
	if err := unprovable(c, text); err != nil {
		return Result{}, err
	}
	started := time.Now()
	schema := scratchPrefix + idgen.Hex(6)
	// The dry run executes what the change would execute: a qualifier naming the connection's own database is pointed at
	// the scratch database, while the mirror below keeps the live names it has to read.
	proved := sqltext.RewriteQualifier(query, c.Database, schema)
	provedRollback := sqltext.RewriteQualifier(rollback, c.Database, schema)

	var mirrored []string
	var summary string
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
		// Cleanup must survive a cancelled request: the schema exists on the server the moment it is created.
		defer func() { _ = dropScratch(context.WithoutCancel(ctx), conn, c.Driver, schema) }()

		refs := verificationRefs(c, proved+"\n"+provedRollback, schema)
		mirrored, err = mirrorReferenced(ctx, conn, c, schema, text)
		if err != nil {
			return struct{}{}, err
		}
		if err := useScratch(ctx, conn, c.Driver, schema); err != nil {
			return struct{}{}, err
		}
		before, err := captureStates(ctx, conn, c.Driver, refs)
		if err != nil {
			return struct{}{}, err
		}
		if _, err := execEach(proved, func(stmt string) error {
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
		if _, err := execEach(provedRollback, func(stmt string) error {
			_, err := conn.ExecContext(ctx, stmt)
			return hintOf(stmt, err)
		}); err != nil {
			return struct{}{}, fmt.Errorf("rollback %w", err)
		}
		after, err := captureStates(ctx, conn, c.Driver, refs)
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

	detail := fmt.Sprintf("verified in scratch schema %q on %s; %s", schema, strings.ToLower(c.Driver), summary)
	if len(mirrored) > 0 {
		detail += "; mirrored " + strings.Join(mirrored, ", ")
	}
	if proved != query || provedRollback != rollback {
		detail += "; qualified names rewritten into the scratch database"
	}
	return Result{
		Affected:      affected,
		DurationMs:    time.Since(started).Milliseconds(),
		Message:       detail,
		ScratchSchema: schema,
	}, nil
}

// qualifiedIdent matches an identifier with a dot-qualified part, including column references that a scratch search_path would not rewrite.
var qualifiedIdent = regexp.MustCompile("(?i)(?:[a-z_][a-z0-9_$]*|\"[^\"]+\"|`[^`]+`)\\s*\\.\\s*(?:[a-z_*][a-z0-9_$]*|\"[^\"]+\"|`[^`]+`)")

// hasQualifiedIdent finds schema.table and table.column references; either can escape a scratch search_path.
func hasQualifiedIdent(text string) bool {
	for _, stmt := range sqltext.SplitStatements(text) {
		if qualifiedIdent.MatchString(stmt) {
			return true
		}
	}
	return false
}

// systemSchemas are readable from a scratch database and are never rewritten: a change may read the catalog.
var systemSchemas = map[string]bool{
	"information_schema": true, "mysql": true, "performance_schema": true, "sys": true,
}

// serverObjects are the objects a statement creates, drops or alters outside any schema.
var serverObjects = map[string]bool{"DATABASE": true, "SCHEMA": true, "USER": true, "ROLE": true, "SERVER": true}

// unprovable refuses a batch a scratch database cannot hold, naming the reason so the queue can group it. Everything
// else is rewritten into the scratch database and proved for real.
func unprovable(c entity.Connection, sqlText string) error {
	for _, stmt := range sqltext.SplitStatements(sqlText) {
		if sqltext.StripLeadingComments(stmt) == "" {
			continue
		}
		if serverScoped(stmt) {
			return interfaces.Unverifiable(constants.UnverifiedServerDDL,
				fmt.Sprintf("%s changes the server rather than a schema, so a scratch database cannot hold it", strings.Join(sqltext.Keywords(stmt, 2), " ")))
		}
		for _, qualifier := range sqltext.Qualifiers(stmt) {
			if strings.EqualFold(qualifier, c.Database) || systemSchemas[strings.ToLower(qualifier)] {
				continue
			}
			return interfaces.Unverifiable(constants.UnverifiedForeignSchema,
				fmt.Sprintf("%q names another database, so the dry run cannot be moved into the scratch database", qualifier))
		}
	}
	return nil
}

// serverScoped reports whether a statement changes the server rather than a schema.
func serverScoped(stmt string) bool {
	words := sqltext.Keywords(stmt, 2)
	if len(words) == 0 {
		return false
	}
	second := ""
	if len(words) > 1 {
		second = words[1]
	}
	switch words[0] {
	case "GRANT", "REVOKE", "FLUSH", "RESET", "PURGE", "INSTALL", "UNINSTALL":
		return true
	case "CREATE", "DROP", "ALTER", "RENAME":
		return serverObjects[second]
	case "SET":
		return second == "GLOBAL" || second == "PERSIST" || second == "PERSIST_ONLY"
	case "CHANGE", "START", "STOP":
		return second == "MASTER" || second == "REPLICA" || second == "SLAVE"
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

// validateChangeTx exercises live tables because the whole transaction is rolled back.
func validateChangeTx(ctx context.Context, c entity.Connection, query, rollback string) (Result, error) {
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

// mirrorReferenced copies the structure of every referenced table that already exists live into the scratch schema, so the change has something to act on.
func mirrorReferenced(ctx context.Context, conn *stdsql.Conn, c entity.Connection, scratch, sqlText string) ([]string, error) {
	created := sqltext.CreatedTables(sqlText)
	refs, _ := sqltext.Referenced(sqlText)
	var mirrored []string
	for _, ref := range refs {
		if created[ref] {
			continue
		}
		schema, table := splitRef(c, ref)
		if !sqltext.IdentOK(schema) || !sqltext.IdentOK(table) || created[schema+"."+table] {
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
		if err := mirrorTableData(ctx, conn, c.Driver, scratch, schema, table); err != nil {
			return nil, fmt.Errorf("copy sample of %s.%s: %w", schema, table, err)
		}
		if err := resetSequences(ctx, conn, c.Driver, scratch, table); err != nil {
			return nil, fmt.Errorf("reset sequence for %s.%s: %w", scratch, table, err)
		}
		mirrored = append(mirrored, schema+"."+table)
	}
	return mirrored, nil
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

// tableExists reports whether a base table exists in the live schema.
func tableExists(ctx context.Context, conn *stdsql.Conn, schema, table string) (bool, error) {
	// identifiers were validated by identOK; placeholders are not portable between pgx and the mysql driver.
	stmt := fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s' AND table_type = 'BASE TABLE'",
		schema, table)
	var n int
	if err := conn.QueryRowContext(ctx, stmt).Scan(&n); err != nil {
		return false, fmt.Errorf("look up %s.%s: %w", schema, table, err)
	}
	return n > 0, nil
}

// mirrorTable creates a structure-only copy (columns, defaults, indexes, constraints) of a live table in the scratch schema.
func mirrorTable(ctx context.Context, conn *stdsql.Conn, driver, scratch, schema, table string) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, fmt.Sprintf(d.mirrorTable, d.quoteRef(scratch, table), d.quoteRef(schema, table)))
	return err
}

// mirrorTableData copies a bounded sample of rows so a DML rollback is tested against data, not an empty shell.
func mirrorTableData(ctx context.Context, conn *stdsql.Conn, driver, scratch, schema, table string) error {
	columns, _, err := copyColumns(ctx, conn, driver, schema, table)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}
	target, source := quoteRef(driver, scratch, table), quoteRef(driver, schema, table)
	list := strings.Join(columns, ", ")
	stmt := "INSERT INTO " + target + " (" + list + ") "
	stmt += "SELECT " + list + " FROM " + source + " LIMIT " + itoa(verifySampleRows)
	_, err = conn.ExecContext(ctx, stmt)
	return err
}

// resetSequences moves generated-id sequences past the copied rows, so an insert that omits the id behaves like live.
func resetSequences(ctx context.Context, conn *stdsql.Conn, driver, scratch, table string) error {
	var column string
	err := conn.QueryRowContext(ctx, `SELECT column_name FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ? AND LOWER(extra) LIKE '%auto_increment%'
		LIMIT 1`, scratch, table).Scan(&column)
	if errors.Is(err, stdsql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	target := quoteRef(driver, scratch, table)
	var next int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(`+quoteColumn(driver, column)+`), 0) + 1 FROM `+target).Scan(&next); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `ALTER TABLE `+target+` AUTO_INCREMENT = `+fmt.Sprint(next))
	return err
}

// copyColumns lists the columns that can be copied into the mirror and whether the table has an identity column.
func copyColumns(ctx context.Context, conn *stdsql.Conn, driver, schema, table string) (columns []string, identity bool, err error) {
	query := `SELECT column_name, extra FROM information_schema.columns
			WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position`
	rows, err := conn.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, first string
		if err := rows.Scan(&name, &first); err != nil {
			return nil, false, err
		}
		extra := strings.ToUpper(first)
		if strings.Contains(extra, "VIRTUAL GENERATED") || strings.Contains(extra, "STORED GENERATED") {
			continue
		}
		columns = append(columns, quoteColumn(driver, name))
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return columns, identity, nil
}

func createScratch(ctx context.Context, conn *stdsql.Conn, driver, schema string) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(d.scratchCreate, schema)); err != nil {
		return fmt.Errorf("create scratch schema: %w", err)
	}
	return nil
}

func dropScratch(ctx context.Context, conn *stdsql.Conn, driver, schema string) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, fmt.Sprintf(d.scratchDrop, schema))
	return err
}

// useScratch points the pinned connection at the scratch schema so unqualified names in the change resolve there.
func useScratch(ctx context.Context, conn *stdsql.Conn, driver, schema string) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(d.scratchUse, schema)); err != nil {
		return fmt.Errorf("select scratch schema: %w", err)
	}
	return nil
}
