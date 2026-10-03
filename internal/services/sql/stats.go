package sql

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"strings"
	"time"

	"noxis/internal/entity"
)

// Stats is a point-in-time snapshot of the instance the console watches.
type Stats struct {
	Driver              string  `json:"driver"`
	Version             string  `json:"version,omitempty"`
	UptimeSeconds       int64   `json:"uptimeSeconds,omitempty"`
	Connections         int64   `json:"connections"`
	MaxConnections      int64   `json:"maxConnections,omitempty"`
	Active              int64   `json:"active"`
	Idle                int64   `json:"idle"`
	IdleInTransaction   int64   `json:"idleInTransaction,omitempty"`
	LongestQuerySeconds float64 `json:"longestQuerySeconds,omitempty"`
	DatabaseSizeBytes   int64   `json:"databaseSizeBytes,omitempty"`
	CacheHitRatio       float64 `json:"cacheHitRatio,omitempty"`
	Commits             int64   `json:"commits,omitempty"`
	Rollbacks           int64   `json:"rollbacks,omitempty"`
	Deadlocks           int64   `json:"deadlocks,omitempty"`
	TempBytes           int64   `json:"tempBytes,omitempty"`
	TakenAt             int64   `json:"takenAt"`
}

// collectStats fills what the driver can answer; each probe fails softly so a
// missing privilege narrows the snapshot instead of failing it.
func collectStats(ctx context.Context, db *stdsql.DB, c entity.Connection) (Stats, error) {
	stats := Stats{Driver: c.Driver, TakenAt: time.Now().Unix()}
	if isMySQL(c.Driver) {
		mysqlStats(ctx, db, &stats)
		return stats, nil
	}
	postgresStats(ctx, db, &stats)
	return stats, nil
}

func postgresStats(ctx context.Context, db *stdsql.DB, s *Stats) {
	scalar(ctx, db, &s.Version, `SELECT version()`)
	scalar(ctx, db, &s.UptimeSeconds, `SELECT extract(epoch FROM now() - pg_postmaster_start_time())::bigint`)
	scalar(ctx, db, &s.MaxConnections, `SELECT setting::bigint FROM pg_settings WHERE name = 'max_connections'`)
	// client backends only: pg_stat_activity also lists background workers,
	// whose state is NULL and which are not "connections" to a user
	const clients = `pg_stat_activity WHERE backend_type = 'client backend'`
	scalar(ctx, db, &s.Connections, `SELECT count(*) FROM `+clients)
	scalar(ctx, db, &s.Active, `SELECT count(*) FROM `+clients+` AND state = 'active' AND pid <> pg_backend_pid()`)
	scalar(ctx, db, &s.Idle, `SELECT count(*) FROM `+clients+` AND state = 'idle'`)
	scalar(ctx, db, &s.IdleInTransaction, `SELECT count(*) FROM `+clients+` AND state = 'idle in transaction'`)
	// the probe's own query is excluded, and clocks can skew a hair negative
	scalar(ctx, db, &s.LongestQuerySeconds, `SELECT coalesce(max(greatest(extract(epoch FROM now() - query_start), 0)), 0)::float8
		FROM `+clients+` AND state = 'active' AND pid <> pg_backend_pid() AND query_start IS NOT NULL`)
	scalar(ctx, db, &s.DatabaseSizeBytes, `SELECT pg_database_size(current_database())`)
	scalar(ctx, db, &s.Commits, `SELECT xact_commit FROM pg_stat_database WHERE datname = current_database()`)
	scalar(ctx, db, &s.Rollbacks, `SELECT xact_rollback FROM pg_stat_database WHERE datname = current_database()`)
	scalar(ctx, db, &s.Deadlocks, `SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()`)
	scalar(ctx, db, &s.TempBytes, `SELECT temp_bytes FROM pg_stat_database WHERE datname = current_database()`)
	scalar(ctx, db, &s.CacheHitRatio, `SELECT CASE WHEN blks_hit + blks_read = 0 THEN 0
		ELSE blks_hit::float8 / (blks_hit + blks_read) END
		FROM pg_stat_database WHERE datname = current_database()`)
}

func mysqlStats(ctx context.Context, db *stdsql.DB, s *Stats) {
	scalar(ctx, db, &s.Version, `SELECT version()`)
	rows := func(q string) map[string]string {
		out := map[string]string{}
		result, err := db.QueryContext(ctx, q)
		if err != nil {
			return out
		}
		defer result.Close()
		for result.Next() {
			var k, v string
			if err := result.Scan(&k, &v); err == nil {
				out[k] = v
			}
		}
		return out
	}
	status := rows(`SHOW GLOBAL STATUS`)
	vars := rows(`SHOW GLOBAL VARIABLES`)
	atoi := func(m map[string]string, k string) int64 {
		var n int64
		for _, r := range m[k] {
			switch {
			case r >= '0' && r <= '9':
				n = n*10 + int64(r-'0')
			default:
				return n
			}
		}
		return n
	}
	s.UptimeSeconds = atoi(status, "Uptime")
	s.Connections = atoi(status, "Threads_connected")
	s.MaxConnections = atoi(vars, "max_connections")
	s.Active = atoi(status, "Threads_running")
	s.Idle = s.Connections - s.Active
	scalar(ctx, db, &s.DatabaseSizeBytes, `SELECT coalesce(sum(data_length + index_length), 0)
		FROM information_schema.tables WHERE table_schema = database()`)
	s.Commits = atoi(status, "Com_commit")
	s.Rollbacks = atoi(status, "Com_rollback")
}

// scalar scans a single-value query into dst, leaving it untouched on error.
func scalar[T any](ctx context.Context, db *stdsql.DB, dst *T, q string) {
	var v T
	if err := db.QueryRowContext(ctx, q).Scan(&v); err == nil {
		*dst = v
	}
}

// estimateRows asks the planner how many rows a read will touch (EXPLAIN
// only — the query is not executed). Zero means "unknown".
func estimateRows(ctx context.Context, db *stdsql.DB, driver, stmt string) int64 {
	q := "EXPLAIN (FORMAT JSON) " + stmt
	if isMySQL(driver) {
		q = "EXPLAIN FORMAT=JSON " + stmt
	}
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

// findRows walks a plan document for the first rows estimate both drivers
// expose ("Plan Rows" for postgres, "rows_examined_per_scan" for mysql).
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
