package sql

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

const indexesQueryMySQL = `
SELECT table_schema, table_name, index_name, non_unique, column_name
FROM information_schema.statistics
WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
ORDER BY table_schema, table_name, index_name, seq_in_index`

// The catalog queries the rest of the browser reads. An engine without one reports nothing: its dialect entry leaves the query empty.
const (
	estimatesQueryPostgres = `
		SELECT n.nspname, c.relname, c.reltuples, pg_total_relation_size(c.oid)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

	estimatesQueryMySQL = `
			SELECT table_schema, table_name, table_rows,
			       COALESCE(data_length, 0) + COALESCE(index_length, 0)
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
			  AND table_type = 'BASE TABLE'`

	commentsQueryPostgres = `
		SELECT n.nspname, c.relname, COALESCE(obj_description(c.oid, 'pg_class'), '')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p', 'v', 'm') AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

	commentsQueryMySQL = `
			SELECT table_schema, table_name, COALESCE(table_comment, '')
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
			  AND table_type = 'BASE TABLE'`
)

// attachIndexes fills each table's indexes; SQLite reads its pragmas per table.
// attachIndexes fills each table's indexes from the catalog; an engine without information_schema reads its pragmas per table instead.
func attachIndexes(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if d.indexesQuery == "" {
		return attachIndexesSQLite(ctx, db, tables)
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

// attachIndexesSQLite reads pragma_index_list and pragma_index_info for each table.
func attachIndexesSQLite(ctx context.Context, db *stdsql.DB, tables []entity.Table) error {
	for i := range tables {
		t := &tables[i]
		list, err := db.QueryContext(ctx, `SELECT name, "unique", origin FROM pragma_index_list(?)`, t.Name)
		if err != nil {
			return fmt.Errorf("indexes of %s: %w", t.Name, err)
		}
		type sqliteIndex struct {
			name    string
			unique  bool
			primary bool
		}
		var found []sqliteIndex
		for list.Next() {
			var name, origin string
			var unique int
			if err := list.Scan(&name, &unique, &origin); err != nil {
				list.Close()
				return err
			}
			found = append(found, sqliteIndex{name: name, unique: unique != 0, primary: origin == "pk"})
		}
		list.Close()
		if err := list.Err(); err != nil {
			return err
		}
		for _, found := range found {
			cols, err := db.QueryContext(ctx, `SELECT name FROM pragma_index_info(?) ORDER BY seqno`, found.name)
			if err != nil {
				return fmt.Errorf("columns of index %s: %w", found.name, err)
			}
			idx := entity.Index{Name: found.name, Unique: found.unique, Primary: found.primary}
			for cols.Next() {
				var col stdsql.NullString
				if err := cols.Scan(&col); err != nil {
					cols.Close()
					return err
				}
				if col.Valid {
					idx.Columns = append(idx.Columns, col.String)
				}
			}
			cols.Close()
			if err := cols.Err(); err != nil {
				return err
			}
			t.Indexes = append(t.Indexes, idx)
		}
	}
	return nil
}

// attachEstimates fills planner row estimates where the catalog reports them; an engine without a statistics catalog reads its pages instead.
func attachEstimates(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if d.estimatesQuery == "" {
		return attachSizesSQLite(ctx, db, tables)
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

// attachSizesSQLite sums dbstat pages when the extension is available; a missing dbstat leaves the size unknown.
func attachSizesSQLite(ctx context.Context, db *stdsql.DB, tables []entity.Table) error {
	for i := range tables {
		var size stdsql.NullInt64
		if err := db.QueryRowContext(ctx, `SELECT SUM(pgsize) FROM dbstat WHERE name = ?`, tables[i].Name).Scan(&size); err != nil {
			continue
		}
		if size.Valid {
			tables[i].SizeBytes = &size.Int64
		}
	}
	return nil
}
