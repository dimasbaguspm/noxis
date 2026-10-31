// Package dragonfly browses and removes Dragonfly keys through typed clients, never raw commands. It serves the dragonfly driver only; every other cache engine is its own package.
package dragonfly

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const (
	driver = constants.DriverDragonfly

	pingTimeout = 10 * time.Second
	opTimeout   = 30 * time.Second

	// maxKeysPerDelete bounds one delete call.
	maxKeysPerDelete = 500

	// maxValueBytes caps a rendered value so a fat string cannot pin the console; the grid says when it was clipped.
	maxValueBytes = 64 * 1024
)

// Result is the shared operation outcome, so the console renders a key's value with the same grid component as a SQL row set.
type Result = entity.Result

// DB talks to Dragonfly servers.
type DB struct{}

// New builds the dragonfly module.
func New() *DB { return &DB{} }

// Satisfier: a dragonfly module browses keys, and nothing else.
var _ interfaces.KeyBrowser = (*DB)(nil)

// Driver is the driver this module serves.
func (db *DB) Driver() string { return driver }

// Supports reports whether this module serves the driver.
func (db *DB) Supports(name string) bool { return strings.EqualFold(name, driver) }

// Ping verifies the server answers and the credentials work.
func (db *DB) Ping(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	_, err = withClient(ctx, c, func(ctx context.Context, cl *goredis.Client) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, pingTimeout)
		defer cancel()
		if err := cl.Ping(ctx).Err(); err != nil {
			return struct{}{}, fmt.Errorf("ping %s: %w", c.Name, err)
		}
		return struct{}{}, nil
	})
	return err
}

// Keys browses by pattern, scanning at most limit keys and resolving each
// key's type, TTL and size in one pipeline.
//
// shortcut: browsing scans at most 5000 keys; add cursor paging when larger keyspaces must be searched.
func (db *DB) Keys(ctx context.Context, c entity.Connection, pattern string, limit int) (page entity.KeyPage, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if pattern == "" {
		pattern = "*"
	}
	if limit <= 0 {
		limit = constants.ScanLimit
	}
	if limit > constants.MaxScanLimit {
		limit = constants.MaxScanLimit
	}
	page.Pattern = pattern
	page.Keys = []entity.Key{}

	_, err = withClient(ctx, c, func(ctx context.Context, cl *goredis.Client) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()

		var cursor uint64
		names := make([]string, 0, limit)
		for len(names) < limit {
			batch, next, err := cl.Scan(ctx, cursor, pattern, int64(limit-len(names))).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("scan %s: %w", c.Name, err)
			}
			names = append(names, batch...)
			cursor = next
			if cursor == 0 {
				break
			}
		}
		if len(names) > limit {
			names = names[:limit]
			page.Truncated = true
		}

		if size, err := cl.DBSize(ctx).Result(); err == nil {
			page.Total = size
		}
		if page.Total > int64(len(names)) {
			page.Truncated = true
		}

		pipe := cl.Pipeline()
		types := make([]*goredis.StatusCmd, len(names))
		ttls := make([]*goredis.DurationCmd, len(names))
		sizes := make([]*goredis.IntCmd, len(names))
		for i, name := range names {
			types[i] = pipe.Type(ctx, name)
			ttls[i] = pipe.TTL(ctx, name)
			sizes[i] = pipe.MemoryUsage(ctx, name)
		}
		// MEMORY USAGE can be denied by ACL or unsupported; only the type and TTL are load-bearing, so pipe errors are not fatal here.
		_, _ = pipe.Exec(ctx)

		for i, name := range names {
			key := entity.Key{Name: name, Type: types[i].Val(), TTL: -1}
			if ttl, err := ttls[i].Result(); err == nil {
				// go-redis reports a missing key as -2 and "no expiry" as -1.
				switch {
				case ttl < 0:
					key.TTL = -1
				default:
					key.TTL = ttl.Milliseconds()
				}
			}
			if size, err := sizes[i].Result(); err == nil {
				key.Size = size
			}
			page.Keys = append(page.Keys, key)
		}
		return struct{}{}, nil
	})
	return page, err
}

// Value renders one key per its type as a two-column grid, reusing the SQL console's result shape.
func (db *DB) Value(ctx context.Context, c entity.Connection, key string) (res Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	started := time.Now()

	_, err = withClient(ctx, c, func(ctx context.Context, cl *goredis.Client) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()

		kind, err := cl.Type(ctx, key).Result()
		if err != nil {
			return struct{}{}, fmt.Errorf("type %s: %w", key, err)
		}
		switch kind {
		case "none":
			return struct{}{}, fmt.Errorf("key %q does not exist", key)
		case "string":
			value, err := cl.Get(ctx, key).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("get %s: %w", key, err)
			}
			res.Columns = []string{"value"}
			res.Rows = [][]string{{clip(value, &res)}}
		case "list":
			values, err := cl.LRange(ctx, key, 0, -1).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("lrange %s: %w", key, err)
			}
			res.Columns = []string{"index", "value"}
			for i, v := range values {
				res.Rows = append(res.Rows, []string{strconv.Itoa(i), clip(v, &res)})
			}
		case "set":
			values, err := cl.SMembers(ctx, key).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("smembers %s: %w", key, err)
			}
			res.Columns = []string{"member"}
			for _, v := range values {
				res.Rows = append(res.Rows, []string{clip(v, &res)})
			}
		case "zset":
			values, err := cl.ZRangeWithScores(ctx, key, 0, -1).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("zrange %s: %w", key, err)
			}
			res.Columns = []string{"score", "member"}
			for _, v := range values {
				res.Rows = append(res.Rows, []string{strconv.FormatFloat(v.Score, 'f', -1, 64), clip(fmt.Sprint(v.Member), &res)})
			}
		case "hash":
			values, err := cl.HGetAll(ctx, key).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("hgetall %s: %w", key, err)
			}
			res.Columns = []string{"field", "value"}
			for field, v := range values {
				res.Rows = append(res.Rows, []string{field, clip(v, &res)})
			}
		case "stream":
			entries, err := cl.XRangeN(ctx, key, "-", "+", 100).Result()
			if err != nil {
				return struct{}{}, fmt.Errorf("xrange %s: %w", key, err)
			}
			res.Columns = []string{"id", "fields"}
			for _, e := range entries {
				fields := make([]string, 0, len(e.Values))
				for f, v := range e.Values {
					fields = append(fields, fmt.Sprintf("%s=%v", f, v))
				}
				res.Rows = append(res.Rows, []string{e.ID, clip(strings.Join(fields, ", "), &res)})
			}
		default:
			return struct{}{}, fmt.Errorf("key %q has unsupported type %q", key, kind)
		}
		res.RowsReturned = len(res.Rows)
		res.Message = "read"
		return struct{}{}, nil
	})
	res.DurationMs = time.Since(started).Milliseconds()
	return res, err
}

// Delete removes keys, returning how many were actually there.
func (db *DB) Delete(ctx context.Context, c entity.Connection, keys []string) (deleted int64, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if len(keys) == 0 {
		return 0, fmt.Errorf("no keys to delete")
	}
	if len(keys) > maxKeysPerDelete {
		return 0, fmt.Errorf("too many keys at once (%d, max %d)", len(keys), maxKeysPerDelete)
	}
	_, err = withClient(ctx, c, func(ctx context.Context, cl *goredis.Client) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		n, err := cl.Del(ctx, keys...).Result()
		if err != nil {
			return struct{}{}, fmt.Errorf("del: %w", err)
		}
		deleted = n
		return struct{}{}, nil
	})
	return deleted, err
}

// Flush empties the connected logical database only.
func (db *DB) Flush(ctx context.Context, c entity.Connection) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	_, err = withClient(ctx, c, func(ctx context.Context, cl *goredis.Client) (struct{}, error) {
		ctx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		if err := cl.FlushDB(ctx).Err(); err != nil {
			return struct{}{}, fmt.Errorf("flushdb %s: %w", c.Name, err)
		}
		return struct{}{}, nil
	})
	return err
}

// withClient opens a client for one operation and closes it; the caller's context rides along so spans and cancellation keep working.
func withClient[T any](ctx context.Context, c entity.Connection, fn func(ctx context.Context, cl *goredis.Client) (T, error)) (T, error) {
	var zero T
	options, err := clientOptions(c)
	if err != nil {
		return zero, err
	}
	cl := goredis.NewClient(options)
	defer func() { _ = cl.Close() }()
	return fn(ctx, cl)
}

func clientOptions(c entity.Connection) (*goredis.Options, error) {
	index := dbIndex(c)
	if index < 0 {
		return nil, fmt.Errorf("connection %q: database %q is not a %s database index", c.Name, c.Database, driver)
	}
	options := &goredis.Options{
		Addr:                  fmt.Sprintf("%s:%d", c.Host, c.Port),
		Username:              c.User,
		Password:              c.Password,
		DB:                    index,
		DialTimeout:           5 * time.Second,
		ReadTimeout:           opTimeout,
		WriteTimeout:          opTimeout,
		ContextTimeoutEnabled: true,
	}
	if c.Params["tls"] == "true" {
		options.TLSConfig = &tls.Config{InsecureSkipVerify: c.Params["insecure_skip_verify"] == "true"}
	}
	return options, nil
}

// dbIndex reads the logical database index from the connection. An empty database means 0; anything else must parse as a non-negative integer.
func dbIndex(c entity.Connection) int {
	if strings.TrimSpace(c.Database) == "" {
		return 0
	}
	index, err := strconv.Atoi(strings.TrimSpace(c.Database))
	if err != nil || index < 0 {
		return -1
	}
	return index
}

// clip caps a rendered value and records the clipping in the message.
func clip(value string, res *Result) string {
	if len(value) <= maxValueBytes {
		return value
	}
	res.Message = fmt.Sprintf("value clipped to %d bytes", maxValueBytes)
	return value[:maxValueBytes] + "…"
}
