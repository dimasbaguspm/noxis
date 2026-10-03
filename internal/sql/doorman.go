// Package sql is the SQL operations core: run reads (query), writes (alter),
// dry-run changes inside a transaction (rollback), refresh planner stats
// (analytics), and introspect schemas — against any registered connection.
// Statement-kind guards run before execute, per statement; the module owns
// all driver/DSN knowledge.
package sql

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"strings"
	"time"

	"noxis/internal/entity"
)

// DB runs verified SQL against registered connections.
type DB struct{}

// New builds the module.
func New() *DB { return &DB{} }

// Result is one operation's outcome: a result grid for reads, an affected
// count/message for writes.
type Result struct {
	Columns  []string   `json:"columns,omitempty"`
	Rows     [][]string `json:"rows,omitempty"`
	Affected int64      `json:"affected"`
	Message  string     `json:"message"`
}

// Ping opens a connection and verifies it answers.
func (u *DB) Ping(ctx context.Context, c entity.Connection) error {
	db, err := open(c)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping %s: %w", c.Name, err)
	}
	return nil
}

// Query runs read-only statements (SELECT/SHOW/EXPLAIN/…; DML and DDL
// rejected — writes belong on alter).
func (u *DB) Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	if err := verifyKind(sqlText, reads); err != nil {
		return Result{}, err
	}
	return u.run(ctx, c, sqlText)
}

// Alter applies writes — DDL (ALTER/CREATE/DROP/…) and DML
// (INSERT/UPDATE/DELETE/…); reads are rejected.
func (u *DB) Alter(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	if err := verifyKind(sqlText, writes); err != nil {
		return Result{}, err
	}
	return u.run(ctx, c, sqlText)
}

// DryRun runs statements inside a transaction that always rolls back inside a transaction that always rolls
// back. On mysql-family drivers DDL is refused: MySQL commits DDL implicitly,
// so a "dry run" there would apply it.
func (u *DB) DryRun(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	if isMySQL(c.Driver) {
		for _, stmt := range splitStatements(sqlText) {
			if kind := firstKeyword(stmt); ddls.set[kind] {
				return Result{}, fmt.Errorf("dry run of %s is not safe on %s (implicit commit) — try it on a postgres connection", kind, c.Driver)
			}
		}
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
	n := 0
	for _, stmt := range splitStatements(sqlText) {
		if firstKeyword(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return Result{}, fmt.Errorf("statement %d: %w", n+1, err)
		}
		n++
	}
	return Result{Message: fmt.Sprintf("rolled back (dry run): %d statement(s)", n)}, nil
}

// Analyze refreshes planner statistics (ANALYZE / ANALYZE TABLE).
func (u *DB) Analyze(ctx context.Context, c entity.Connection, table string) (Result, error) {
	if !identOK(table) {
		return Result{}, fmt.Errorf("invalid table name %q", table)
	}
	stmt := "ANALYZE " + table
	if isMySQL(c.Driver) {
		stmt = "ANALYZE TABLE " + table
	}
	return u.run(ctx, c, stmt)
}

// run executes each statement separately (verified incrementally, no
// multi-statement driver quirks) and returns the last statement's outcome.
func (u *DB) run(ctx context.Context, c entity.Connection, sqlText string) (Result, error) {
	db, err := open(c)
	if err != nil {
		return Result{}, err
	}
	defer db.Close()

	var last Result
	n := 0
	for _, stmt := range splitStatements(sqlText) {
		kind := firstKeyword(stmt)
		if kind == "" {
			continue
		}
		n++
		if reads.set[kind] {
			if last, err = query(ctx, db, stmt); err != nil {
				return Result{}, fmt.Errorf("statement %d: %w", n, err)
			}
			continue
		}
		if last, err = exec(ctx, db, stmt); err != nil {
			return Result{}, fmt.Errorf("statement %d: %w", n, err)
		}
	}
	if n == 0 {
		return Result{}, fmt.Errorf("no statements to run")
	}
	last.Message = fmt.Sprintf("%d statement(s) executed", n)
	return last, nil
}

// open builds a small pool per operation (control-plane rates).
func open(c entity.Connection) (*stdsql.DB, error) {
	name, err := sqlDriver(c.Driver)
	if err != nil {
		return nil, err
	}
	conn, err := dsn(c)
	if err != nil {
		return nil, err
	}
	db, err := stdsql.Open(name, conn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", c.Name, err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func query(ctx context.Context, db *stdsql.DB, stmt string) (Result, error) {
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return Result{}, err
	}
	res := Result{Columns: cols}
	for rows.Next() {
		cells := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range cells {
			ptrs[i] = &cells[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return Result{}, err
		}
		row := make([]string, len(cols))
		for i, cell := range cells {
			row[i] = stringify(cell)
		}
		res.Rows = append(res.Rows, row)
	}
	return res, rows.Err()
}

func exec(ctx context.Context, db *stdsql.DB, stmt string) (Result, error) {
	out, err := db.ExecContext(ctx, stmt)
	if err != nil {
		return Result{}, err
	}
	n, err := out.RowsAffected()
	if err != nil {
		n = 0 // DDL reports no count everywhere
	}
	return Result{Affected: n}, nil
}

// stringify renders a driver value for the console grid.
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	case time.Time:
		return x.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// VerifyWrite exposes the write guard (a change's query must be a write).
func VerifyWrite(sqlText string) error { return verifyKind(sqlText, writes) }

// VerifyRead exposes the read guard.
func VerifyRead(sqlText string) error { return verifyKind(sqlText, reads) }

// verifyKind rejects every statement whose first keyword is not allowed.
// The check runs per statement — split outside quoted strings and comments —
// so DDL cannot be smuggled behind a read in a multi-statement batch
// (`SELECT 1; DROP TABLE x` is rejected, not executed). Empty fragments
// (trailing semicolons) are ignored.
func verifyKind(sqlText string, allowed kindSet) error {
	for _, stmt := range splitStatements(sqlText) {
		kind := firstKeyword(stmt)
		if kind == "" {
			continue
		}
		if !allowed.set[kind] {
			return fmt.Errorf("statement %q not allowed here (only: %s)", kind, allowed.names)
		}
	}
	return nil
}

// splitStatements splits on ';' while skipping string literals ('...' with
// ” escapes, "..." for identifiers, `...` for mysql) and comments (-- and
// /* */), so semicolons inside values do not split the batch.
//
// ponytail: backslash-escaped quotes inside literals (mysql without
// NO_BACKSLASH_ESCAPES) can still desync the splitter — console statements
// rarely carry them; a real SQL lexer if that ever matters.
func splitStatements(sqlText string) []string {
	var stmts []string
	var cur strings.Builder
	i, n := 0, len(sqlText)
	for i < n {
		c := sqlText[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			q := c
			cur.WriteByte(c)
			i++
			for i < n {
				cur.WriteByte(sqlText[i])
				if sqlText[i] == q {
					if i+1 < n && sqlText[i+1] == q { // doubled quote = escaped
						cur.WriteByte(sqlText[i+1])
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '-' && i+1 < n && sqlText[i+1] == '-':
			cur.WriteString("--")
			i += 2
			for i < n && sqlText[i] != '\n' {
				cur.WriteByte(sqlText[i])
				i++
			}
		case c == '/' && i+1 < n && sqlText[i+1] == '*':
			cur.WriteString("/*")
			i += 2
			for i < n && (sqlText[i] != '*' || i+1 >= n || sqlText[i+1] != '/') {
				cur.WriteByte(sqlText[i])
				i++
			}
			if i+1 < n {
				cur.WriteString("*/")
				i += 2
			}
		case c == ';':
			stmts = append(stmts, cur.String())
			cur.Reset()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}
	return append(stmts, cur.String())
}

func firstKeyword(sqlText string) string {
	trimmed := strings.TrimSpace(sqlText)
	if i := strings.IndexAny(trimmed, " \t\r\n"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return strings.ToUpper(strings.Trim(trimmed, "(;"))
}

// identOK allows a plain or schema-qualified table name (identifier safety
// for ANALYZE, which cannot use a placeholder).
func identOK(s string) bool {
	if s == "" {
		return false
	}
	for _, part := range strings.Split(s, ".") {
		if part == "" {
			return false
		}
		for _, r := range part {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			default:
				return false
			}
		}
	}
	return true
}
