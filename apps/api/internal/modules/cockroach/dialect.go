package cockroach

import (
	stdsql "database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	// database/sql drivers, registered for the drivers we accept.
	_ "github.com/jackc/pgx/v5/stdlib"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// This file is the one place this driver's dialect is known: the wire driver, the quote character, the default schema, the catalog queries and the verification rules. Another driver is another module with its own copy of this file, by decision.

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

var postgresBaseline = dialect{
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
	tableExists:      `SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = '%s' AND c.relname = '%s' AND c.relkind IN ('r', 'p', 'v', 'm')`,
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

// profile is this driver's dialect, derived from the postgres baseline in this package: DDL is transactional, so a change is proved in the live transaction, and there is no CREATE TABLE (LIKE ... INCLUDING ALL), so the scratch path must never run.
var profile = func() dialect {
	d := postgresBaseline
	d.name = constants.DriverCockroach
	d.txn = txnAlways
	// cockroach keeps its own statistics catalog and range byte counters, so the estimate query is its own; it has no
	// EXPLAIN (FORMAT JSON), so the planner path stays off. The rest of the postgres catalog works unchanged. The scratch
	// fields it inherits are never read: nothing in this package creates a throwaway schema.
	d.estimatesQuery = estimatesQueryCockroach
	d.explain = ""
	return d
}()

// dialectOf resolves the one driver this package serves to its profile; every other driver is refused.
func dialectOf(driver string) (dialect, error) {
	if !strings.EqualFold(driver, constants.DriverCockroach) {
		return dialect{}, fmt.Errorf("driver %q is not served by the cockroach module", driver)
	}
	return profile, nil
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

// quoteRef quotes a table reference.
func (d dialect) quoteRef(schema, table string) string {
	q := string(d.quote)
	if schema != "" {
		return q + schema + q + "." + q + table + q
	}
	return q + table + q
}

// quoteRef is the driver-aware identifier quoting the console and the verifier share.
func quoteRef(driver, schema, table string) string {
	if d, err := dialectOf(driver); err == nil {
		return d.quoteRef(schema, table)
	}
	return `"` + schema + `"."` + table + `"`
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

// scanPostgresEstimate reads reltuples and the relation size.
func scanPostgresEstimate(rows *stdsql.Rows) (estimateRow, error) {
	var r estimateRow
	err := rows.Scan(&r.schema, &r.table, &r.estimate, &r.size)
	return r, err
}
