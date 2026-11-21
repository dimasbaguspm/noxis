package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"noxis/internal/common/ref"
	"noxis/internal/common/sqltext"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
	"sort"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// scratchPrefix names the throwaway database a shadow rehearsal runs in. The suffix is random, so
// two rehearsals against the same server never collide.
const scratchPrefix = "noxis_rehearse_"

// sampleRows bounds how much data a `rehearsal` run copies. It is a sample, and the note says so:
// a row count from a sample proves the statement's shape, not its effect on the whole table.
const sampleRows = 200

// Rehearse proves a change in a database it creates and drops. MySQL and MariaDB commit DDL
// implicitly, so a transaction proves nothing for a schema change: the only honest proof is to
// rebuild the schema somewhere else, run the statement there, and check that the rollback puts it
// back.
func (d *DB) Rehearse(ctx context.Context, target domain.Target, password string, change domain.Change) (out domain.Rehearsal, err error) {
	started := time.Now()
	ctx, end := telemetry.Span(ctx, "mysql.DB.Rehearse")
	defer func() { end(err) }()
	tier := change.Tier
	if tier == "" {
		tier = domain.TierShadow
	}
	rehearsal := domain.Rehearsal{Tier: tier, CreatedAt: started.UTC()}

	scope, err := rehearsalScope(change, target.Database)
	if err != nil {
		rehearsal.Status = domain.RehearsalUnsupported
		rehearsal.Error = err.Error()
		rehearsal.Note = "the statement could not be scoped to this target, so nothing was run"
		rehearsal.DurationMS = time.Since(started).Milliseconds()
		return rehearsal, nil
	}

	pool, err := d.pool(target, password)
	if err != nil {
		return rehearsal, err
	}
	defer func() { _ = pool.Close() }()

	if tier == domain.TierStatic {
		d.rehearseStatic(ctx, pool, target, password, change, scope, &rehearsal)
		rehearsal.DurationMS = time.Since(started).Milliseconds()
		return rehearsal, nil
	}
	if degraded := d.rehearseInScratch(ctx, pool, target.Database, change, scope, &rehearsal); degraded {
		// A server that will not let the target create a database can still be checked: parse, plan and
		// estimate, and say by name that the proof is the weaker one.
		reason := rehearsal.Error
		rehearsal = domain.Rehearsal{Tier: domain.TierStatic, CreatedAt: rehearsal.CreatedAt}
		d.rehearseStatic(ctx, pool, target, password, change, scope, &rehearsal)
		rehearsal.Note += "; " + staticInsteadNote(reason)
	}
	rehearsal.DurationMS = time.Since(started).Milliseconds()
	return rehearsal, nil
}

// rehearsalScope names the tables a change and its rollback touch, and refuses a statement aimed at
// another schema: rewriting it would run the change somewhere the reviewer never agreed to.
func rehearsalScope(change domain.Change, database string) ([]string, error) {
	statement, err := sqltext.Classify(change.Query)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrInvalid, err)
	}
	names := append([]string(nil), statement.Tables...)
	if strings.TrimSpace(change.Rollback) != "" {
		if rollback, err := sqltext.Classify(change.Rollback); err == nil {
			names = append(names, rollback.Tables...)
		}
	}

	seen := map[string]bool{}
	var scope []string
	for _, name := range names {
		qualified := sqltext.ParseQualified(name)
		if qualified.Database != "" && !strings.EqualFold(qualified.Database, database) {
			return nil, fmt.Errorf("%w: the statement names schema %q, and this target governs %q",
				domain.ErrUnsupported, qualified.Database, database)
		}
		if qualified.Table != "" && !seen[qualified.Table] {
			seen[qualified.Table] = true
			scope = append(scope, qualified.Table)
		}
	}
	sort.Strings(scope)
	return scope, nil
}

// rehearseStatic parses, plans and estimates without touching anything. It can never claim more
// than `partial`: nothing was executed, so nothing was proven.
func (*DB) rehearseStatic(ctx context.Context, pool *sql.DB, target domain.Target, password string,
	change domain.Change, scope []string, rehearsal *domain.Rehearsal,
) {
	rehearsal.Status = domain.RehearsalPartial
	rehearsal.Irreversible = irreversible(change)
	rehearsal.Note = "static: the statement was parsed and planned, nothing was executed, so the reversal is unproven"

	var notes []string
	if plan, err := staticPlan(ctx, pool, change.Query); err != nil {
		notes = append(notes, "EXPLAIN unavailable: "+scrub(err, password))
	} else if plan != "" {
		rehearsal.Plan = plan
	}
	if estimate, err := tableEstimates(ctx, pool, target.Database, scope); err != nil {
		notes = append(notes, "no estimate: "+scrub(err, password))
	} else if estimate != "" {
		rehearsal.RowDiff = estimate
	}
	if len(notes) > 0 {
		rehearsal.Note += "; " + strings.Join(notes, "; ")
	}
	if rehearsal.Irreversible {
		rehearsal.Note += "; no rollback was given, so this change is irreversible"
	}
}

// rehearseInScratch rebuilds the schema in a throwaway database, runs the change, and requires the
// rollback to put the schema back exactly as it was.
func (*DB) rehearseInScratch(ctx context.Context, pool *sql.DB, database string,
	change domain.Change, scope []string, rehearsal *domain.Rehearsal,
) (degraded bool) {
	scratch := scratchPrefix + strings.ReplaceAll(ref.New(), "-", "")[:12]

	conn, err := pool.Conn(ctx)
	if err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		return
	}
	defer func() { _ = conn.Close() }()

	if err := createScratch(ctx, conn, scratch); err != nil {
		rehearsal.Error = err.Error()
		if errors.Is(err, errNoPrivilege) {
			// The caller degrades to a static check: the reason travels in Error, and this function
			// leaves the status for that caller to set.
			return true
		}
		rehearsal.Status = domain.RehearsalUnsupported
		rehearsal.Note = "the rehearsal needs CREATE DATABASE on the target server; without it only tier=static can run"
		return
	}
	// The drop has to survive a cancelled request: the database exists on the server the moment it
	// is created.
	defer func() { dropScratch(context.WithoutCancel(ctx), conn, scratch) }()

	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		return
	}
	if err := mirrorTables(ctx, conn, database, scratch, scope); err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		rehearsal.Note = "the throwaway database could not be built from the tables this change names"
		return
	}
	if change.Tier == domain.TierRehearsal {
		if err := copyRows(ctx, conn, database, scratch, scope); err != nil {
			rehearsal.Status = domain.RehearsalFailed
			rehearsal.Error = err.Error()
			return
		}
	}

	before, err := snapshotSchema(ctx, conn, scratch)
	if err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		return
	}
	beforeCounts, _ := tableCounts(ctx, conn, scratch)

	rewritten := sqltext.RewriteQualifier(change.Query, database, scratch)
	if _, err := conn.ExecContext(ctx, rewritten); err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = scrub(err, "")
		rehearsal.Note = "the statement itself failed in the throwaway database, so there is nothing to approve"
		return
	}

	after, err := snapshotSchema(ctx, conn, scratch)
	if err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		return
	}
	rehearsal.SchemaDiff = diffSchemas(before, after)
	rehearsal.RowDiff = rowDiff(ctx, conn, scratch, beforeCounts, change.Tier)

	rehearsal.Irreversible = irreversible(change)
	if strings.TrimSpace(change.Rollback) == "" {
		rehearsal.Status = domain.RehearsalPartial
		rehearsal.Note = "the change ran in a rebuilt schema, but no rollback was given, so the reversal is unproven"
		return
	}

	if _, err := conn.ExecContext(ctx, sqltext.RewriteQualifier(change.Rollback, database, scratch)); err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = scrub(err, "")
		rehearsal.Note = "the rollback did not run, so the change cannot be undone as written"
		return
	}
	restored, err := snapshotSchema(ctx, conn, scratch)
	if err != nil {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		return
	}
	if !sameSchema(before, restored) {
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.SchemaDiff += "\n--- after the rollback\n" + diffSchemas(after, restored)
		rehearsal.Note = "the rollback left the schema different from where it started"
		return
	}

	rehearsal.Status = domain.RehearsalVerified
	rehearsal.Note = "rebuilt from the engine's own DDL in a throwaway database; the rollback restored it exactly"
	if change.Tier != domain.TierRehearsal {
		rehearsal.Note += "; no rows were copied, so a DML result here proves shape, not effect, " +
			"and an index is proven to build, not to hold on your data"
	}
	return false
}

// staticPlan asks the engine how it would run the statement. DDL has no plan, which is not a
// failure: it is a fact about DDL.
func staticPlan(ctx context.Context, pool *sql.DB, statement string) (string, error) {
	if classified, err := sqltext.Classify(statement); err != nil || classified.Kind == sqltext.DDL {
		return "", nil
	}
	var plan string
	if err := pool.QueryRowContext(ctx, "EXPLAIN FORMAT=JSON "+statement).Scan(&plan); err != nil {
		return "", err
	}
	return plan, nil
}

// tableEstimates reports the engine's own row and size estimate for the tables in scope.
func tableEstimates(ctx context.Context, pool *sql.DB, database string, scope []string) (string, error) {
	if len(scope) == 0 {
		return "", nil
	}
	var out strings.Builder
	for _, table := range scope {
		var rows, size int64
		err := pool.QueryRowContext(ctx, `SELECT COALESCE(table_rows, 0),
				COALESCE(data_length, 0) + COALESCE(index_length, 0)
			FROM information_schema.tables WHERE table_schema = ? AND table_name = ?`, database, table).
			Scan(&rows, &size)
		if err != nil {
			return out.String(), err
		}
		fmt.Fprintf(&out, "%s: ~%d rows, %d bytes (engine estimate)\n", table, rows, size)
	}
	return out.String(), nil
}

func createScratch(ctx context.Context, conn *sql.Conn, scratch string) error {
	if _, err := conn.ExecContext(ctx, "CREATE DATABASE "+quote(scratch)); err != nil {
		if refusedForPrivilege(err) {
			return fmt.Errorf("%w: %w", errNoPrivilege, err)
		}
		return fmt.Errorf("%w: %w", domain.ErrUnsupported, err)
	}
	return nil
}

func dropScratch(ctx context.Context, conn *sql.Conn, scratch string) {
	_, _ = conn.ExecContext(ctx, "DROP DATABASE IF EXISTS "+quote(scratch))
}

// mirrorTables rebuilds each table in scope from SHOW CREATE TABLE. The statement the engine prints
// names the table bare, so running it inside the throwaway database creates it there.
func mirrorTables(ctx context.Context, conn *sql.Conn, database, scratch string, scope []string) error {
	if _, err := conn.ExecContext(ctx, "USE "+quote(scratch)); err != nil {
		return err
	}
	for _, table := range scope {
		var ddl string
		err := conn.QueryRowContext(ctx,
			fmt.Sprintf("SHOW CREATE TABLE %s.%s", quote(database), quote(table))).Scan(new(string), &ddl)
		if err != nil {
			return fmt.Errorf("%w: table %s: %w", domain.ErrNotFound, table, err)
		}
		if _, err := conn.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("%w: rebuilding %s: %w", domain.ErrUnsupported, table, err)
		}
	}
	return nil
}

// copyRows samples data for a `rehearsal` tier run. A copy is bounded: the note says so, because a
// row count from a sample is not the row count.
func copyRows(ctx context.Context, conn *sql.Conn, database, scratch string, scope []string) error {
	if _, err := conn.ExecContext(ctx, "USE "+quote(scratch)); err != nil {
		return err
	}
	for _, table := range scope {
		//nolint:gosec // every identifier is quoted by quote(); nothing here comes from a client
		statement := "INSERT INTO " + quote(table) + " SELECT * FROM " + quote(database) + "." + quote(table) +
			" LIMIT " + strconv.Itoa(sampleRows)
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%w: sampling %s: %w", domain.ErrUnsupported, table, err)
		}
	}
	return nil
}

// snapshotSchema reads every base table's DDL in one schema. Comparing two snapshots is how a
// rehearsal proves a change and its reversal.
func snapshotSchema(ctx context.Context, conn *sql.Conn, schema string) (map[string]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT table_name FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	snapshot := make(map[string]string, len(names))
	for _, name := range names {
		var ddl string
		if err := conn.QueryRowContext(ctx,
			fmt.Sprintf("SHOW CREATE TABLE %s.%s", quote(schema), quote(name))).Scan(new(string), &ddl); err != nil {
			return nil, err
		}
		snapshot[name] = normaliseDDL(ddl)
	}
	return snapshot, nil
}

// tableCounts answers how many rows each table in the throwaway database holds.
func tableCounts(ctx context.Context, conn *sql.Conn, schema string) (map[string]int64, error) {
	rows, err := conn.QueryContext(ctx, `SELECT table_name FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(names))
	for _, name := range names {
		var count int64
		if err := conn.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quote(schema), quote(name))).Scan(&count); err != nil {
			return nil, err
		}
		counts[name] = count
	}
	return counts, nil
}

// rowDiff reports what a run did to row counts, or says plainly that it looked at none.
func rowDiff(ctx context.Context, conn *sql.Conn, schema string, before map[string]int64, tier domain.Tier) string {
	after, err := tableCounts(ctx, conn, schema)
	if err != nil {
		return "row counts unavailable: " + err.Error()
	}
	var out strings.Builder
	for name, count := range after {
		fmt.Fprintf(&out, "%s: %d -> %d rows\n", name, before[name], count)
	}
	if len(after) == 0 {
		out.WriteString("no tables left in the throwaway database\n")
	}
	if tier != domain.TierRehearsal {
		out.WriteString("(no rows were copied: tier=rehearsal samples up to " +
			strconv.Itoa(sampleRows) + " rows per table)\n")
	}
	return out.String()
}

// normaliseDDL removes what changes on its own — the AUTO_INCREMENT counter — so a snapshot compares
// the shape of a table and not its history.
func normaliseDDL(ddl string) string {
	lines := strings.Split(ddl, "\n")
	for i, line := range lines {
		if at := strings.Index(line, " AUTO_INCREMENT="); at >= 0 {
			rest := line[at+len(" AUTO_INCREMENT="):]
			end := strings.IndexByte(rest, ' ')
			if end < 0 {
				line = line[:at]
			} else {
				line = line[:at] + rest[end:]
			}
		}
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

// diffSchemas prints what changed between two snapshots, one line per difference.
func diffSchemas(before, after map[string]string) string {
	var out strings.Builder
	for _, name := range unionKeys(before, after) {
		old, hadOld := before[name]
		fresh, hasFresh := after[name]
		switch {
		case !hasFresh:
			fmt.Fprintf(&out, "removed table %s\n", name)
			for _, line := range strings.Split(old, "\n") {
				fmt.Fprintf(&out, "- %s\n", line)
			}
		case !hadOld:
			fmt.Fprintf(&out, "added table %s\n", name)
			for _, line := range strings.Split(fresh, "\n") {
				fmt.Fprintf(&out, "+ %s\n", line)
			}
		case old != fresh:
			fmt.Fprintf(&out, "changed table %s\n", name)
			out.WriteString(lineDiff(old, fresh))
		}
	}
	if out.Len() == 0 {
		return "no schema change"
	}
	return out.String()
}

func lineDiff(before, after string) string {
	oldLines := strings.Split(before, "\n")
	freshLines := strings.Split(after, "\n")
	known := make(map[string]bool, len(oldLines))
	for _, line := range oldLines {
		known[line] = true
	}
	fresh := make(map[string]bool, len(freshLines))
	for _, line := range freshLines {
		fresh[line] = true
	}
	var out strings.Builder
	for _, line := range oldLines {
		if !fresh[line] {
			fmt.Fprintf(&out, "- %s\n", line)
		}
	}
	for _, line := range freshLines {
		if !known[line] {
			fmt.Fprintf(&out, "+ %s\n", line)
		}
	}
	return out.String()
}

func sameSchema(before, after map[string]string) bool {
	if len(before) != len(after) {
		return false
	}
	for name, ddl := range before {
		if after[name] != ddl {
			return false
		}
	}
	return true
}

func unionKeys(a, b map[string]string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var keys []string
	for key := range a {
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	for key := range b {
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// irreversible reports whether a change has no way back: a DML or DDL with no rollback, or no
// rollback at all for a statement that plainly destroys something.
func irreversible(change domain.Change) bool {
	if strings.TrimSpace(change.Rollback) != "" {
		return false
	}
	statement, err := sqltext.Classify(change.Query)
	if err != nil {
		return true
	}
	switch statement.Kind {
	case sqltext.Insert, sqltext.DDL, sqltext.Delete, sqltext.Update:
		return true
	}
	return false
}

// errNoPrivilege marks the one shadow failure a static check survives: the server refused to create
// the throwaway database, so nothing ran there and nothing could have. Its text is short because it is
// always read inside a sentence that already says what could not be created.
var errNoPrivilege = errors.New("no privilege")

// refusedForPrivilege reports whether the server refused the database itself, rather than the
// connection or the statement. A wrong password is 1045 and is deliberately not one of these:
// nothing connected, so nothing can be claimed about the change.
func refusedForPrivilege(err error) bool {
	var driverErr *mysqldriver.MySQLError
	if !errors.As(err, &driverErr) {
		return false
	}
	return driverErr.Number == 1044 || driverErr.Number == 1142
}

// staticInsteadNotePattern is the sentence a degraded rehearsal says about itself: the engine's own
const staticInsteadNotePattern = "the throwaway database could not be created (%s), so this is only a static " +
	"check; grant CREATE DATABASE to the target's user for the shadow rehearsal"

// staticInsteadNote fills the reason in.
func staticInsteadNote(reason string) string {
	return strings.Replace(staticInsteadNotePattern, "%s", reason, 1)
}
