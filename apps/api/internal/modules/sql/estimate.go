package sql

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"strings"
)

// estimateRows asks the planner how many rows a read will touch (EXPLAIN only — the query is not executed).
func estimateRows(ctx context.Context, db *stdsql.DB, driver, stmt string) int64 {
	d, err := dialectOf(driver)
	if err != nil || d.explain == "" {
		// an engine that reports access paths instead of row counts has no estimate to give
		return 0
	}
	q := d.explain + stmt
	var raw string
	if err := db.QueryRowContext(ctx, q).Scan(&raw); err != nil {
		return 0
	}
	var doc any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return 0
	}
	return findRows(doc)
}

// findRows walks a plan document for the first rows estimate both drivers expose ("Plan Rows" for postgres, "rows_examined_per_scan" for mysql).
func findRows(node any) int64 {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			switch key {
			case "Plan Rows", "rows_examined_per_scan":
				if f, ok := toFloat(child); ok {
					return int64(f)
				}
			}
		}
		for _, child := range v {
			if rows := findRows(child); rows > 0 {
				return rows
			}
		}
	case []any:
		for _, child := range v {
			if rows := findRows(child); rows > 0 {
				return rows
			}
		}
	}
	return 0
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := json.Number(strings.TrimSpace(n)).Float64()
		return f, err == nil
	}
	return 0, false
}
