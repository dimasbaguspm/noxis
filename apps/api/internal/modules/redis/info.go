package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// Info reads what the server says about itself: memory, eviction, the hit ratio, clients and the per-database key counts.
func (db *DB) Info(ctx context.Context, c entity.Connection) (info entity.CacheInfo, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	_, err = withClient(ctx, c, func(ctx context.Context, cl *goredis.Client) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		raw, err := cl.Info(ctx, "server", "clients", "memory", "stats", "keyspace").Result()
		if err != nil {
			return struct{}{}, fmt.Errorf("info %s: %w", c.Name, err)
		}
		info = parseInfo(raw)
		return struct{}{}, nil
	})
	return info, err
}

// parseInfo reads the INFO reply: `field:value` per line, sections introduced by a `# name` line, and one `dbN:…` line
// per logical database.
func parseInfo(raw string) entity.CacheInfo {
	var info entity.CacheInfo
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch key {
		case "redis_version", "valkey_version", "dragonfly_version":
			info.Version = value
		case "uptime_in_seconds":
			info.UptimeSeconds = parseInfoInt(value)
		case "used_memory":
			info.UsedMemoryBytes = parseInfoInt(value)
		case "used_memory_peak":
			info.PeakMemoryBytes = parseInfoInt(value)
		case "maxmemory":
			info.MaxMemoryBytes = parseInfoInt(value)
		case "evicted_keys":
			info.EvictedKeys = parseInfoInt(value)
		case "expired_keys":
			info.ExpiredKeys = parseInfoInt(value)
		case "keyspace_hits":
			info.KeyspaceHits = parseInfoInt(value)
		case "keyspace_misses":
			info.KeyspaceMisses = parseInfoInt(value)
		case "connected_clients":
			info.ConnectedClients = parseInfoInt(value)
		case "instantaneous_ops_per_sec":
			info.OpsPerSecond = parseInfoInt(value)
		default:
			if strings.HasPrefix(key, "db") {
				info.Databases = append(info.Databases, parseKeyspace(key, value))
			}
		}
	}
	return info
}

// parseKeyspace reads one `db0:keys=3,expires=1,avg_ttl=0` line.
func parseKeyspace(name, value string) entity.CacheDatabase {
	database := entity.CacheDatabase{Name: name}
	for _, part := range strings.Split(value, ",") {
		field, raw, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch field {
		case "keys":
			database.Keys = parseInfoInt(raw)
		case "expires":
			database.Expires = parseInfoInt(raw)
		}
	}
	return database
}

// parseInfoInt reads a counter the server reports as text; a field it omits reads as absent rather than invented.
func parseInfoInt(raw string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
