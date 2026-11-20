package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// systemSchemas are left out of the listing: they are the server's own bookkeeping, and a reader
// who wants them can still name them in a query.
var systemSchemas = []string{"information_schema", "mysql", "performance_schema", "sys"}

// pool opens one connection pool for one call and hands it to fn, so no caller can leak a pool.
func (*DB) pool(target domain.Target, password string) (*sql.DB, error) {
	pool, err := sql.Open(driverName, dsnOf(target, password))
	if err != nil {
		return nil, statementError(err, password)
	}
	pool.SetMaxOpenConns(1)
	return pool, nil
}

// Query runs one statement a read route has already classified. It reads at most limit rows and
// says whether it stopped early; it never rewrites the statement.
func (d *DB) Query(ctx context.Context, target domain.Target, password, statement string, limit int) (out domain.Rows, err error) {
	started := time.Now()
	ctx, end := telemetry.Span(ctx, "mysql.DB.Query")
	defer func() { end(err) }()
	pool, err := d.pool(target, password)
	if err != nil {
		return domain.Rows{}, err
	}
	defer func() { _ = pool.Close() }()

	rows, err := pool.QueryContext(ctx, statement)
	if err != nil {
		return domain.Rows{}, statementError(err, password)
	}
	defer func() { _ = rows.Close() }()

	types, err := rows.ColumnTypes()
	if err != nil {
		return domain.Rows{}, statementError(err, password)
	}
	columns := make([]domain.Column, 0, len(types))
	for _, column := range types {
		columns = append(columns, domain.Column{Name: column.Name(), Type: column.DatabaseTypeName()})
	}

	values := make([][]*string, 0, limit)
	truncated := false
	for rows.Next() {
		if len(values) == limit {
			// One row past the ceiling is all the evidence needed that more exist.
			truncated = true
			break
		}
		row := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range row {
			dest[i] = &row[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return domain.Rows{}, statementError(err, password)
		}
		values = append(values, stringsOf(row))
	}
	if err := rows.Err(); err != nil {
		return domain.Rows{}, statementError(err, password)
	}

	return domain.Rows{
		Columns:    columns,
		Values:     values,
		RowCount:   len(values),
		Truncated:  truncated,
		DurationMS: time.Since(started).Milliseconds(),
	}, nil
}

// Databases lists the schemas the connection can see, minus the server's own.
func (d *DB) Databases(ctx context.Context, target domain.Target, password string) ([]string, error) {
	pool, err := d.pool(target, password)
	if err != nil {
		return nil, err
	}
	defer func() { _ = pool.Close() }()

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(systemSchemas)), ",")
	args := make([]any, len(systemSchemas))
	for i, name := range systemSchemas {
		args[i] = name
	}
	//nolint:gosec // placeholders is a run of "?" built here; the names travel as arguments
	rows, err := pool.QueryContext(ctx,
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN ("+placeholders+
			") ORDER BY schema_name", args...)
	if err != nil {
		return nil, statementError(err, password)
	}
	defer func() { _ = rows.Close() }()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, statementError(err, password)
		}
		databases = append(databases, name)
	}
	return databases, statementError(rows.Err(), password)
}

// Tables lists the tables in one schema, with the engine's own row and size estimates.
func (d *DB) Tables(ctx context.Context, target domain.Target, password, database string) ([]domain.Table, error) {
	pool, err := d.pool(target, password)
	if err != nil {
		return nil, err
	}
	defer func() { _ = pool.Close() }()

	rows, err := pool.QueryContext(ctx, `SELECT table_name, COALESCE(engine, ''), COALESCE(table_rows, 0),
			COALESCE(data_length, 0) + COALESCE(index_length, 0)
		FROM information_schema.tables WHERE table_schema = ? ORDER BY table_name`, database)
	if err != nil {
		return nil, statementError(err, password)
	}
	defer func() { _ = rows.Close() }()

	var tables []domain.Table
	for rows.Next() {
		var table domain.Table
		if err := rows.Scan(&table.Name, &table.Engine, &table.Rows, &table.SizeBytes); err != nil {
			return nil, statementError(err, password)
		}
		tables = append(tables, table)
	}
	return tables, statementError(rows.Err(), password)
}

// Table describes one table. The DDL comes from SHOW CREATE TABLE, which is the statement a
// rehearsal will rebuild the table from, so the reviewer and the rehearsal read the same text.
func (d *DB) Table(ctx context.Context, target domain.Target, password, database, table string) (domain.TableDetail, error) {
	pool, err := d.pool(target, password)
	if err != nil {
		return domain.TableDetail{}, err
	}
	defer func() { _ = pool.Close() }()

	detail := domain.TableDetail{Table: domain.Table{Name: table}}
	err = pool.QueryRowContext(ctx, `SELECT COALESCE(engine, ''), COALESCE(table_rows, 0),
			COALESCE(data_length, 0) + COALESCE(index_length, 0)
		FROM information_schema.tables WHERE table_schema = ? AND table_name = ?`, database, table).
		Scan(&detail.Engine, &detail.Rows, &detail.SizeBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TableDetail{}, fmt.Errorf("%w: table %s.%s", domain.ErrNotFound, database, table)
	}
	if err != nil {
		return domain.TableDetail{}, statementError(err, password)
	}

	if detail.Columns, err = d.columns(ctx, pool, password, database, table); err != nil {
		return domain.TableDetail{}, err
	}
	if detail.Indexes, err = d.indexes(ctx, pool, password, database, table); err != nil {
		return domain.TableDetail{}, err
	}

	var create string
	createErr := pool.QueryRowContext(ctx, fmt.Sprintf("SHOW CREATE TABLE %s.%s", quote(database), quote(table))).
		Scan(new(string), &create)
	if createErr != nil {
		return domain.TableDetail{}, statementError(createErr, password)
	}
	detail.CreateSQL = create
	return detail, nil
}

func (*DB) columns(ctx context.Context, pool *sql.DB, password, database, table string) ([]domain.ColumnInfo, error) {
	rows, err := pool.QueryContext(ctx, `SELECT column_name, column_type, is_nullable, column_key, extra, column_default
		FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position`,
		database, table)
	if err != nil {
		return nil, statementError(err, password)
	}
	defer func() { _ = rows.Close() }()

	var columns []domain.ColumnInfo
	for rows.Next() {
		var (
			column   domain.ColumnInfo
			nullable string
			def      *string
		)
		if err := rows.Scan(&column.Name, &column.Type, &nullable, &column.Key, &column.Extra, &def); err != nil {
			return nil, statementError(err, password)
		}
		column.Nullable = strings.EqualFold(nullable, "YES")
		column.Default = def
		columns = append(columns, column)
	}
	return columns, statementError(rows.Err(), password)
}

func (*DB) indexes(ctx context.Context, pool *sql.DB, password, database, table string) ([]domain.IndexInfo, error) {
	rows, err := pool.QueryContext(ctx, `SELECT index_name, non_unique, column_name
		FROM information_schema.statistics WHERE table_schema = ? AND table_name = ?
		ORDER BY index_name, seq_in_index`, database, table)
	if err != nil {
		return nil, statementError(err, password)
	}
	defer func() { _ = rows.Close() }()

	var (
		indexes []domain.IndexInfo
		last    string
	)
	for rows.Next() {
		var (
			name    string
			nonUniq int
			column  *string
		)
		if err := rows.Scan(&name, &nonUniq, &column); err != nil {
			return nil, statementError(err, password)
		}
		if len(indexes) == 0 || name != last {
			indexes = append(indexes, domain.IndexInfo{Name: name, Unique: nonUniq == 0})
			last = name
		}
		if column != nil {
			indexes[len(indexes)-1].Columns = append(indexes[len(indexes)-1].Columns, *column)
		}
	}
	return indexes, statementError(rows.Err(), password)
}

// quote wraps an identifier in the engine's own quoting, doubling any backtick inside it. Every
// schema or table name from a path goes through here; nothing is concatenated raw.
func quote(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

// stringsOf renders one scanned row as text, keeping NULL as nil instead of an empty string.
func stringsOf(row []any) []*string {
	out := make([]*string, len(row))
	for i, value := range row {
		if value == nil {
			continue
		}
		var text string
		switch typed := value.(type) {
		case []byte:
			text = string(typed)
		case string:
			text = typed
		case time.Time:
			text = typed.UTC().Format(time.RFC3339Nano)
		default:
			text = fmt.Sprint(typed)
		}
		value := text
		out[i] = &value
	}
	return out
}

// statementError maps what the engine said onto what the caller should do about it, and cuts the
// password out of the message in case a driver echoes a DSN back.
func statementError(err error, password string) error {
	if err == nil {
		return nil
	}
	message := scrub(err, password)
	var engine *mysqldriver.MySQLError
	if errors.As(err, &engine) {
		switch engine.Number {
		case 1049, 1146: // unknown database, unknown table
			return fmt.Errorf("%w: %s", domain.ErrNotFound, message)
		case 1054, 1064: // unknown column, syntax error
			return fmt.Errorf("%w: %s", domain.ErrInvalid, message)
		case 1044: // access denied to a schema
			return fmt.Errorf("%w: %s", domain.ErrForbidden, message)
		}
	}
	return fmt.Errorf("%w: %s", domain.ErrUnsupported, message)
}

// scrub removes the password from anything the driver says.
func scrub(err error, password string) string {
	message := err.Error()
	if password != "" {
		message = strings.ReplaceAll(message, password, "***")
	}
	return message
}
