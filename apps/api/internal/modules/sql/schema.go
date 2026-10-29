package sql

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"noxis/internal/entity"
)

// sysSchemaFilter excludes engine-internal schemas (valid for postgres and mysql).
const sysSchemaFilter = `table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')`

// listTables reports every browsable table with its columns and declared keys, so the schema browser and the ERD view read one pass per driver.
func listTables(ctx context.Context, db *stdsql.DB, driver string) ([]entity.Table, error) {
	d, err := dialectOf(driver)
	if err != nil {
		return nil, err
	}
	if d.catalogPerTable {
		tables, err := listTablesSQLite(ctx, db)
		if err != nil {
			return nil, err
		}
		if err := listKeysSQLite(ctx, db, tables); err != nil {
			return nil, err
		}
		if err := attachIndexes(ctx, db, driver, tables); err != nil {
			return nil, err
		}
		if err := attachEstimates(ctx, db, driver, tables); err != nil {
			return nil, err
		}
		return tables, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT table_schema, table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE `+sysSchemaFilter+`
		ORDER BY table_schema, table_name, ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("list schema: %w", err)
	}
	defer rows.Close()
	var (
		out []entity.Table
		cur *entity.Table
	)
	for rows.Next() {
		var schema, table, name, typ, nullable string
		if err := rows.Scan(&schema, &table, &name, &typ, &nullable); err != nil {
			return nil, err
		}
		if cur == nil || cur.Schema != schema || cur.Name != table {
			out = append(out, entity.Table{Schema: schema, Name: table})
			cur = &out[len(out)-1]
		}
		cur.Columns = append(cur.Columns, entity.Column{Name: name, Type: typ, Nullable: nullable == "YES"})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := attachKeys(ctx, db, driver, out); err != nil {
		return nil, err
	}
	if err := attachIndexes(ctx, db, driver, out); err != nil {
		return nil, err
	}
	if err := attachEstimates(ctx, db, driver, out); err != nil {
		return nil, err
	}
	if err := attachComments(ctx, db, driver, out); err != nil {
		return nil, err
	}
	return out, nil
}

// listTablesSQLite lists main-schema tables and their columns.
func listTablesSQLite(ctx context.Context, db *stdsql.DB) ([]entity.Table, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list schema: %w", err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]entity.Table, 0, len(names))
	for _, name := range names {
		cols, err := db.QueryContext(ctx,
			`SELECT name, type, "notnull", pk FROM pragma_table_info(?) ORDER BY cid`, name)
		if err != nil {
			return nil, fmt.Errorf("table %s: %w", name, err)
		}
		t := entity.Table{Schema: schemaSQLite, Name: name}
		for cols.Next() {
			var col, typ string
			var notnull, pk int
			if err := cols.Scan(&col, &typ, &notnull, &pk); err != nil {
				cols.Close()
				return nil, err
			}
			t.Columns = append(t.Columns, entity.Column{Name: col, Type: typ, Nullable: notnull == 0, PrimaryKey: pk > 0})
		}
		cols.Close()
		if err := cols.Err(); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}
