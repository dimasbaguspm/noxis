package sql

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"noxis/internal/entity"
)

// Table is a browsable relation with its columns.
type Table struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// Column is one relation attribute.
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// Tables lists user relations and their columns (information_schema works
// for both supported drivers).
func (u *DB) Tables(ctx context.Context, c entity.Connection) ([]Table, error) {
	db, err := open(c)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tables, err := listTables(ctx, db, c)
	if err != nil {
		return nil, err
	}
	byKey := map[string]*Table{}
	for i := range tables {
		byKey[tables[i].Schema+"."+tables[i].Name] = &tables[i]
	}
	rows, err := db.QueryContext(ctx, `
		SELECT table_schema, table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE `+sysSchemaFilter+`
		ORDER BY ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var schema, table, name, typ, nullable string
		if err := rows.Scan(&schema, &table, &name, &typ, &nullable); err != nil {
			return nil, err
		}
		if t, ok := byKey[schema+"."+table]; ok {
			t.Columns = append(t.Columns, Column{Name: name, Type: typ, Nullable: nullable == "YES"})
		}
	}
	return tables, rows.Err()
}

func listTables(ctx context.Context, db *stdsql.DB, c entity.Connection) ([]Table, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE' AND `+sysSchemaFilter+`
		ORDER BY table_schema, table_name`)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()
	var out []Table
	for rows.Next() {
		var t Table
		if err := rows.Scan(&t.Schema, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// sysSchemaFilter excludes engine-internal schemas.
const sysSchemaFilter = `table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys')`
