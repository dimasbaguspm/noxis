package sqlite

import (
	stdsql "database/sql"
	"fmt"
	"net/url"
	"strings"

	// database/sql drivers, registered for the drivers we accept.
	_ "modernc.org/sqlite"

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

// profile is this driver's dialect: one connection, pragmas instead of information_schema, and every batch proved by rolling back the live transaction.
var profile = dialect{
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
	tableExists:     `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = '%[2]s'`,
	mirrorTable:     "",
	generatedColumn: "GENERATED",
}

// dialectOf resolves the one driver this package serves to its profile; every other driver is refused.
func dialectOf(driver string) (dialect, error) {
	if !strings.EqualFold(driver, constants.DriverSQLite) {
		return dialect{}, fmt.Errorf("driver %q is not served by the sqlite module", driver)
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
