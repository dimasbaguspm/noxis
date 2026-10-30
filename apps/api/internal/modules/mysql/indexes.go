package mysql

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"noxis/internal/entity"
)

const indexesQueryMySQL = `
SELECT table_schema, table_name, index_name, non_unique, column_name
FROM information_schema.statistics
WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
ORDER BY table_schema, table_name, index_name, seq_in_index`

// The catalog queries the rest of the browser reads. An engine without one reports nothing: its dialect entry leaves the query empty.
const (
	estimatesQueryMySQL = `
			SELECT table_schema, table_name, table_rows,
			       COALESCE(data_length, 0) + COALESCE(index_length, 0)
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
			  AND table_type = 'BASE TABLE'`
	commentsQueryMySQL = `
			SELECT table_schema, table_name, COALESCE(table_comment, '')
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
			  AND table_type = 'BASE TABLE'`
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
