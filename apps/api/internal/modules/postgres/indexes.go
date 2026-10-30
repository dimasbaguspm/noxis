package postgres

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"noxis/internal/entity"
)

// indexesQueryPostgres reads every index and its key columns in declaration order.
const indexesQueryPostgres = `
SELECT n.nspname, t.relname, ic.relname, ix.indisunique, ix.indisprimary, a.attname
FROM pg_index ix
JOIN pg_class t ON t.oid = ix.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_class ic ON ic.oid = ix.indexrelid
JOIN generate_subscripts(ix.indkey::int2[], 1) AS s(ord) ON s.ord <= ix.indnkeyatts
JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = (ix.indkey::int2[])[s.ord]
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
ORDER BY n.nspname, t.relname, ic.relname, s.ord`

// The catalog queries the rest of the browser reads. An engine without one reports nothing: its dialect entry leaves the query empty.
const (
	estimatesQueryPostgres = `
		SELECT n.nspname, c.relname, c.reltuples, pg_total_relation_size(c.oid)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema')`
	commentsQueryPostgres = `
		SELECT n.nspname, c.relname, COALESCE(obj_description(c.oid, 'pg_class'), '')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p', 'v', 'm') AND n.nspname NOT IN ('pg_catalog', 'information_schema')`
)

// attachIndexes fills each table's indexes from the catalog.
func attachIndexes(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if d.indexesQuery == "" {
		return nil
	}
	rows, err := db.QueryContext(ctx, d.indexesQuery)
	if err != nil {
		return fmt.Errorf("list indexes: %w", err)
	}
	defer rows.Close()
	index := tableIndex(tables)
	last, lastName := -1, ""
	for rows.Next() {
		row, err := d.scanIndex(rows)
		if err != nil {
			return err
		}
		i, ok := index[refKey(row.schema, row.table)]
		if !ok {
			continue
		}
		if last != i || lastName != row.name {
			tables[i].Indexes = append(tables[i].Indexes, entity.Index{Name: row.name, Unique: row.unique, Primary: row.primary})
			last, lastName = i, row.name
		}
		if row.column.Valid {
			tables[i].Indexes[len(tables[i].Indexes)-1].Columns = append(tables[i].Indexes[len(tables[i].Indexes)-1].Columns, row.column.String)
		}
	}
	return rows.Err()
}

// attachEstimates fills planner row estimates where the catalog reports them; an engine without a statistics catalog reads its pages instead.
func attachEstimates(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if d.estimatesQuery == "" {
		return nil
	}
	rows, err := db.QueryContext(ctx, d.estimatesQuery)
	if err != nil {
		return fmt.Errorf("list row estimates: %w", err)
	}
	defer rows.Close()
	index := tableIndex(tables)
	for rows.Next() {
		row, err := d.scanEstimate(rows)
		if err != nil {
			return err
		}
		i, ok := index[refKey(row.schema, row.table)]
		if !ok {
			continue
		}
		if row.estimate.Valid && row.estimate.Float64 >= 0 {
			n := int64(row.estimate.Float64)
			tables[i].EstimatedRows = &n
		}
		if row.size.Valid {
			tables[i].SizeBytes = &row.size.Int64
		}
	}
	return rows.Err()
}

// attachComments fills the table comment from the catalog; an engine without table comments reports none.
func attachComments(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if d.commentsQuery == "" {
		return nil
	}
	rows, err := db.QueryContext(ctx, d.commentsQuery)
	if err != nil {
		return fmt.Errorf("list table comments: %w", err)
	}
	defer rows.Close()
	index := tableIndex(tables)
	for rows.Next() {
		var schema, table, comment string
		if err := rows.Scan(&schema, &table, &comment); err != nil {
			return err
		}
		if i, ok := index[refKey(schema, table)]; ok {
			tables[i].Comment = comment
		}
	}
	return rows.Err()
}
