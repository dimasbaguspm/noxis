package sqlite

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"noxis/internal/entity"
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

// listKeysSQLite reads SQLite's own catalog: pragma_foreign_key_list carries the declared references (pragma_table_info's pk flag is applied by the caller).
func listKeysSQLite(ctx context.Context, db *stdsql.DB, tables []entity.Table) error {
	for i := range tables {
		t := &tables[i]
		rows, err := db.QueryContext(ctx, `SELECT "from", "table", "to" FROM pragma_foreign_key_list(?)`, t.Name)
		if err != nil {
			return fmt.Errorf("foreign keys of %s: %w", t.Name, err)
		}
		for rows.Next() {
			var column, refTable, refColumn string
			if err := rows.Scan(&column, &refTable, &refColumn); err != nil {
				rows.Close()
				return err
			}
			if col := findColumn(t, column); col != nil {
				col.References = &entity.Reference{Schema: "main", Table: refTable, Column: refColumn}
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return nil
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
