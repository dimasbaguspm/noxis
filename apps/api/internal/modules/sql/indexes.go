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

// attachIndexes fills each table's indexes; SQLite reads its pragmas per table.
func attachIndexes(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	if isSQLite(driver) {
		return attachIndexesSQLite(ctx, db, tables)
	}
	query := indexesQueryPostgres
	if isMySQL(driver) {
		query = indexesQueryMySQL
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("list indexes: %w", err)
	}
	defer rows.Close()
	index := tableIndex(tables)
	last, lastName := -1, ""
	for rows.Next() {
		var schema, table, name string
		var unique, primary bool
		var column stdsql.NullString
		if isMySQL(driver) {
			var nonUnique int
			if err := rows.Scan(&schema, &table, &name, &nonUnique, &column); err != nil {
				return err
			}
			unique, primary = nonUnique == 0, name == "PRIMARY"
		} else if err := rows.Scan(&schema, &table, &name, &unique, &primary, &column); err != nil {
			return err
		}
		i, ok := index[refKey(schema, table)]
		if !ok {
			continue
		}
		if last != i || lastName != name {
			tables[i].Indexes = append(tables[i].Indexes, entity.Index{Name: name, Unique: unique, Primary: primary})
			last, lastName = i, name
		}
		if column.Valid {
			tables[i].Indexes[len(tables[i].Indexes)-1].Columns = append(tables[i].Indexes[len(tables[i].Indexes)-1].Columns, column.String)
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

// attachEstimates fills planner row estimates where the catalog reports them; SQLite has no statistics catalog.
func attachEstimates(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	if isSQLite(driver) {
		return attachSizesSQLite(ctx, db, tables)
	}
	query := `
		SELECT n.nspname, c.relname, c.reltuples, pg_total_relation_size(c.oid)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema')`
	if isMySQL(driver) {
		query = `
			SELECT table_schema, table_name, table_rows,
			       COALESCE(data_length, 0) + COALESCE(index_length, 0)
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
			  AND table_type = 'BASE TABLE'`
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("list row estimates: %w", err)
	}
	defer rows.Close()
	index := tableIndex(tables)
	for rows.Next() {
		var schema, table string
		var estimate stdsql.NullFloat64
		var size stdsql.NullInt64
		if isMySQL(driver) {
			var n stdsql.NullInt64
			if err := rows.Scan(&schema, &table, &n, &size); err != nil {
				return err
			}
			if n.Valid {
				estimate = stdsql.NullFloat64{Float64: float64(n.Int64), Valid: true}
			}
		} else if err := rows.Scan(&schema, &table, &estimate, &size); err != nil {
			return err
		}
		if i, ok := index[refKey(schema, table)]; ok {
			if estimate.Valid && estimate.Float64 >= 0 {
				n := int64(estimate.Float64)
				tables[i].EstimatedRows = &n
			}
			if size.Valid {
				tables[i].SizeBytes = &size.Int64
			}
		}
	}
	return rows.Err()
}

// attachComments fills the table comment from the catalog; SQLite has no native table comments.
func attachComments(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	if isSQLite(driver) {
		return nil
	}
	query := `
		SELECT n.nspname, c.relname, COALESCE(obj_description(c.oid, 'pg_class'), '')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p', 'v', 'm') AND n.nspname NOT IN ('pg_catalog', 'information_schema')`
	if isMySQL(driver) {
		query = `
			SELECT table_schema, table_name, COALESCE(table_comment, '')
			FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
			  AND table_type = 'BASE TABLE'`
	}
	rows, err := db.QueryContext(ctx, query)
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
