package cockroach

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
