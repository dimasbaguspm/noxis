package mysql

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"time"
)

func query(ctx context.Context, db *stdsql.DB, stmt string) (Result, error) {
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return Result{}, err
	}
	res := Result{Columns: cols}
	for rows.Next() {
		cells := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range cells {
			ptrs[i] = &cells[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return Result{}, err
		}
		row := make([]string, len(cols))
		for i, cell := range cells {
			row[i] = stringify(cell)
		}
		res.Rows = append(res.Rows, row)
	}
	return res, rows.Err()
}

func exec(ctx context.Context, db *stdsql.DB, stmt string) (Result, error) {
	out, err := db.ExecContext(ctx, stmt)
	if err != nil {
		return Result{}, err
	}
	n, err := out.RowsAffected()
	if err != nil {
		n = 0 // DDL reports no count everywhere
	}
	return Result{Affected: n}, nil
}

// stringify renders a driver value for the console grid.
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	case time.Time:
		return x.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", x)
	}
}
