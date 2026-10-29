package sql

import (
	stdsql "database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"

	// database/sql drivers, registered for the drivers we accept.
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// This file is the one place a database's dialect is known. Everything that differs between the engines noxis serves — the wire driver, the quote character, the default schema, the catalog queries, the verification rules — is one entry in dialectByName, so a feature reads a field instead of asking which driver it is on and a new engine is an entry, not an edit in every file.

// txnRule says when a batch is proved by rolling back the live transaction instead of a throwaway schema.
type txnRule int

const (
	// txnAlways: the engine has transactional DDL, so every batch is proved in the live transaction.
	txnAlways txnRule = iota
	// txnWhenQualified: a qualified name escapes a scratch schema, so those batches use the live transaction and the rest use scratch.
	txnWhenQualified
	// txnWithoutDDL: DDL commits implicitly, so only DML can be proved in the live transaction; DDL needs scratch.
	txnWithoutDDL
)

// indexRow is one index column as the catalog reports it, normalized so the reader never sees two shapes.
type indexRow struct {
	schema  string
	table   string
	name    string
	unique  bool
	primary bool
	column  stdsql.NullString
}

// estimateRow is one table's planner estimate and size, normalized the same way.
type estimateRow struct {
	schema   string
	table    string
	estimate stdsql.NullFloat64
	size     stdsql.NullInt64
}

// commentRow is one table's comment.
type commentRow struct {
	schema  string
	table   string
	comment string
}

// dialect is everything that differs between the databases noxis serves.
type dialect struct {
	// name is the driver name the console and the stored record use.
	name string
	// driver is the database/sql driver that speaks it.
	driver string
	// quote is the identifier quote character.
	quote byte
	// schema is the schema a bare table name resolves to; empty means the connection's own database.
	schema string
	// dsn builds the driver-native connection string.
	dsn func(entity.Connection) (string, error)
	// maxOpenConns and maxIdleConns size the pool: an engine that serializes writers gains nothing from a second connection.
	maxOpenConns int
	maxIdleConns int
	// catalogPerTable reports a catalog with no information_schema, where each table is read through pragmas instead.
	catalogPerTable bool

	// analyze is the ANALYZE statement for one table, with room for the name.
	analyze string
	// explain reads a row estimate from the planner as a JSON document; empty means the engine reports none.
	explain string

	// txn says when a batch is proved by rolling back the live transaction.
	txn txnRule
	// refuseQualifiedDDL rejects a DDL batch that names a schema: the scratch schema would not contain it, so the dry run would lie.
	refuseQualifiedDDL bool
	// scratchCreate, scratchDrop and scratchUse create, drop and enter the throwaway schema, with room for its name.
	scratchCreate string
	scratchDrop   string
	scratchUse    string
	// tableExists reports whether a base table exists, from the catalog rather than the data.
	tableExists string
	// mirrorTable creates a structure-only copy, with room for the target and then the source reference.
	mirrorTable string
	// identityOverride is the INSERT clause that lets an explicit value into an identity column.
	identityOverride string
	// copyColumnsQuery lists the columns a mirror can copy, with room for the schema and then the table.
	copyColumnsQuery string
	// generatedColumn lists the metadata that marks a column as generated, so a mirror never copies one.
	generatedColumn string

	// catalog queries. An empty query means the engine does not report that.
	keysQuery      string
	indexesQuery   string
	estimatesQuery string
	commentsQuery  string
	// scanIndex and scanEstimate normalize each catalog row, because the engines return different columns for the same fact.
	scanIndex    func(*stdsql.Rows) (indexRow, error)
	scanEstimate func(*stdsql.Rows) (estimateRow, error)
}

// dialectByName is the catalog. One entry per accepted driver name, so a driver cannot be half-added: dialect_test enumerates the constants and fails when an entry or a field is missing.
var dialectByName = map[string]dialect{
	constants.DriverPostgres: postgresDialect,
	"postgresql":             postgresDialect, // the long form of the same engine
	constants.DriverMySQL:    mysqlDialect,
	constants.DriverMariaDB:  mysqlDialect, // same protocol, same catalog, a different server
	constants.DriverSQLite:   sqliteDialect,
}

var postgresDialect = dialect{
	maxOpenConns: 4,
	maxIdleConns: 2,
	name:         constants.DriverPostgres,
	driver:       dbDriverPostgres,
	quote:        '"',
	schema:       schemaPostgres,
	dsn:          dsnPostgres,

	analyze: "ANALYZE %s",
	explain: "EXPLAIN (FORMAT JSON) ",

	txn:              txnWhenQualified,
	scratchCreate:    `CREATE SCHEMA "%s"`,
	scratchDrop:      `DROP SCHEMA IF EXISTS "%s" CASCADE`,
	scratchUse:       `SET search_path TO "%s", public`,
	tableExists:      `SELECT to_regclass('%s.%s')::text`,
	mirrorTable:      `CREATE TABLE %s (LIKE %s INCLUDING ALL)`,
	identityOverride: "OVERRIDING SYSTEM VALUE ",
	copyColumnsQuery: `SELECT column_name, is_generated, is_identity FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`,
	generatedColumn: "ALWAYS",

	keysQuery:      keysQueryPostgres,
	indexesQuery:   indexesQueryPostgres,
	estimatesQuery: estimatesQueryPostgres,
	commentsQuery:  commentsQueryPostgres,
	scanIndex:      scanPostgresIndex,
	scanEstimate:   scanPostgresEstimate,
}

var mysqlDialect = dialect{
	maxOpenConns: 4,
	maxIdleConns: 2,
	name:         constants.DriverMySQL,
	driver:       dbDriverMySQL,
	quote:        '`',
	schema:       "", // a bare name resolves to the connection's own database
	dsn:          dsnMySQL,

	analyze: "ANALYZE TABLE %s",
	explain: "EXPLAIN FORMAT=JSON ",

	txn:                txnWithoutDDL,
	refuseQualifiedDDL: true,
	scratchCreate:      "CREATE DATABASE `%s`",
	scratchDrop:        "DROP DATABASE IF EXISTS `%s`",
	scratchUse:         "USE `%s`",
	tableExists:        `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s'`,
	mirrorTable:        "CREATE TABLE %s LIKE %s",
	identityOverride:   "",
	copyColumnsQuery: `SELECT column_name, extra FROM information_schema.columns
			WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position`,
	generatedColumn: "GENERATED",

	keysQuery:      keysQueryMySQL,
	indexesQuery:   indexesQueryMySQL,
	estimatesQuery: estimatesQueryMySQL,
	commentsQuery:  commentsQueryMySQL,
	scanIndex:      scanMySQLIndex,
	scanEstimate:   scanMySQLEstimate,
}

var sqliteDialect = dialect{
	name:            constants.DriverSQLite,
	driver:          dbDriverSQLite,
	quote:           '"',
	schema:          schemaSQLite,
	catalogPerTable: true,
	maxOpenConns:    1,
	dsn:             dsnSQLite,

	analyze: "ANALYZE %s",
	explain: "", // EXPLAIN QUERY PLAN reports access paths, not row counts

	txn:             txnAlways,
	tableExists:     `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = '%s'`,
	mirrorTable:     "",
	generatedColumn: "GENERATED",
}

// dialectOf resolves a driver name to its profile.
func dialectOf(driver string) (dialect, error) {
	d, ok := dialectByName[strings.ToLower(driver)]
	if !ok {
		return dialect{}, fmt.Errorf("unsupported driver %q (postgres, mysql or sqlite)", driver)
	}
	return d, nil
}

// sqlDriver maps a driver name to a database/sql driver.
func sqlDriver(driver string) (string, error) {
	d, err := dialectOf(driver)
	if err != nil {
		return "", err
	}
	return d.driver, nil
}

// analyzeStmt is the ANALYZE statement for one table.
func (d dialect) analyzeStmt(table string) string { return fmt.Sprintf(d.analyze, table) }

// transactional reports whether the batch is proved by rolling back the live transaction instead of a scratch schema.
func (d dialect) transactional(text string) bool {
	switch d.txn {
	case txnAlways:
		return true
	case txnWhenQualified:
		return hasQualifiedIdent(text)
	case txnWithoutDDL:
		return !containsDDL(text)
	}
	return false
}

// qualifiedDDLRefused reports whether this engine must refuse DDL with qualified names instead of dry-running it.
func (d dialect) qualifiedDDLRefused(text string) bool {
	return d.refuseQualifiedDDL && containsDDL(text) && hasQualifiedIdent(text)
}

// quoteRef quotes a table reference.
func (d dialect) quoteRef(schema, table string) string {
	q := string(d.quote)
	if schema != "" {
		return q + schema + q + "." + q + table + q
	}
	return q + table + q
}

// quoteColumn quotes a column name.
func (d dialect) quoteColumn(name string) string {
	q := string(d.quote)
	return q + strings.ReplaceAll(name, q, q+q) + q
}

// quoteRef is the driver-aware identifier quoting the console and the verifier share.
func quoteRef(driver, schema, table string) string {
	if d, err := dialectOf(driver); err == nil {
		return d.quoteRef(schema, table)
	}
	return `"` + schema + `"."` + table + `"`
}

func quoteColumn(driver, name string) string {
	if d, err := dialectOf(driver); err == nil {
		return d.quoteColumn(name)
	}
	return `"` + name + `"`
}

// tablePresentStatement is the catalog lookup for one table, with identifiers inlined because the two drivers do not share placeholder syntax.
func (d dialect) tablePresentStatement(schema, table string) string {
	return fmt.Sprintf(d.tableExists, schema, table)
}

// dsnPostgres is the URL form pgx accepts; sslmode stays the homelab default unless the connection says otherwise.
func dsnPostgres(c entity.Connection) (string, error) {
	u := &url.URL{
		Scheme: dsnSchemePostgres,
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   "/" + c.Database,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	for k, v := range c.Params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// dsnMySQL is the DSN the mysql driver parses, so no string is hand-assembled.
func dsnMySQL(c entity.Connection) (string, error) {
	mc := mysql.NewConfig()
	mc.User, mc.Passwd = c.User, c.Password
	mc.Net, mc.Addr = "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	mc.DBName = c.Database
	mc.ParseTime = true
	mc.Params = c.Params
	return mc.FormatDSN(), nil
}

// dsnSQLite is the database file path; any extra params become a file: URI.
func dsnSQLite(c entity.Connection) (string, error) {
	if len(c.Params) == 0 {
		return c.Database, nil
	}
	q := url.Values{}
	for k, v := range c.Params {
		q.Set(k, v)
	}
	return "file:" + c.Database + "?" + q.Encode(), nil
}

// dsn builds the driver-native connection string.
func dsn(c entity.Connection) (string, error) {
	d, err := dialectOf(c.Driver)
	if err != nil {
		return "", err
	}
	return d.dsn(c)
}

// scanPostgresIndex reads pg_index's columns: uniqueness and primacy are booleans in the catalog.
func scanPostgresIndex(rows *stdsql.Rows) (indexRow, error) {
	var r indexRow
	err := rows.Scan(&r.schema, &r.table, &r.name, &r.unique, &r.primary, &r.column)
	return r, err
}

// scanMySQLIndex reads information_schema.statistics, which reports uniqueness as a flag and names the primary key "PRIMARY".
func scanMySQLIndex(rows *stdsql.Rows) (indexRow, error) {
	var r indexRow
	var nonUnique int
	if err := rows.Scan(&r.schema, &r.table, &r.name, &nonUnique, &r.column); err != nil {
		return r, err
	}
	r.unique, r.primary = nonUnique == 0, r.name == "PRIMARY"
	return r, nil
}

// scanPostgresEstimate reads reltuples and the relation size.
func scanPostgresEstimate(rows *stdsql.Rows) (estimateRow, error) {
	var r estimateRow
	err := rows.Scan(&r.schema, &r.table, &r.estimate, &r.size)
	return r, err
}

// scanMySQLEstimate reads table_rows, an integer count, and the summed sizes.
func scanMySQLEstimate(rows *stdsql.Rows) (estimateRow, error) {
	var r estimateRow
	var n stdsql.NullInt64
	if err := rows.Scan(&r.schema, &r.table, &n, &r.size); err != nil {
		return r, err
	}
	if n.Valid {
		r.estimate = stdsql.NullFloat64{Float64: float64(n.Int64), Valid: true}
	}
	return r, nil
}
