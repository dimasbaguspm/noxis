package cockroach

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"noxis/internal/entity"
)

// keysQuery reads every declared key on a table: primary keys, unique constraints, and foreign keys with the column they reference.
const (
	keysQueryPostgres = `
	SELECT tc.table_schema, tc.table_name, kcu.column_name, tc.constraint_type,
	       ccu.table_schema, ccu.table_name, ccu.column_name
	FROM information_schema.table_constraints tc
	JOIN information_schema.key_column_usage kcu
	  ON kcu.constraint_name = tc.constraint_name AND kcu.constraint_schema = tc.constraint_schema
	LEFT JOIN information_schema.constraint_column_usage ccu
	  ON ccu.constraint_name = tc.constraint_name AND ccu.constraint_schema = tc.constraint_schema
	WHERE tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE', 'FOREIGN KEY')
	  AND tc.table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')
	ORDER BY tc.table_schema, tc.table_name, kcu.ordinal_position`
)

// attachKeys fills the key flags of tables already listed by listTables.
func attachKeys(ctx context.Context, db *stdsql.DB, driver string, tables []entity.Table) error {
	d, err := dialectOf(driver)
	if err != nil {
		return err
	}
	if d.keysQuery == "" {
		return nil
	}
	rows, err := db.QueryContext(ctx, d.keysQuery)
	if err != nil {
		return fmt.Errorf("list keys: %w", err)
	}
	defer rows.Close()
	index := tableIndex(tables)
	for rows.Next() {
		var schema, table, column, kind string
		var refSchema, refTable, refColumn stdsql.NullString
		if err := rows.Scan(&schema, &table, &column, &kind, &refSchema, &refTable, &refColumn); err != nil {
			return err
		}
		i, ok := index[refKey(schema, table)]
		if !ok {
			continue
		}
		col := findColumn(&tables[i], column)
		if col == nil {
			continue
		}
		switch kind {
		case "PRIMARY KEY":
			col.PrimaryKey = true
		case "UNIQUE":
			col.Unique = true
		case "FOREIGN KEY":
			if refSchema.Valid && refTable.Valid && refColumn.Valid {
				col.References = &entity.Reference{
					Schema: refSchema.String, Table: refTable.String, Column: refColumn.String,
				}
			}
		}
	}
	return rows.Err()
}

// tableIndex maps a schema-qualified table to its position, so the key pass does not rescan the list per key.
func tableIndex(tables []entity.Table) map[string]int {
	index := make(map[string]int, len(tables))
	for i := range tables {
		index[refKey(tables[i].Schema, tables[i].Name)] = i
	}
	return index
}

// refKey is the identity of a table in the key pass.
func refKey(schema, table string) string { return schema + "." + table }

// findColumn returns the named column, or nil when the catalog reported a key on a column the column pass did not list.
func findColumn(t *entity.Table, name string) *entity.Column {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i]
		}
	}
	return nil
}
