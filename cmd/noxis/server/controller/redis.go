package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"noxis/internal/entity"
)

// The Redis namespace. Connection CRUD shares one implementation with SQL
// (the module decides which drivers belong to it); what is Redis-only is the
// key browser, the value inspector, key removal and INFO stats.
//
// Writes stop at removing keys: a Redis value cannot be rolled back, so no
// key mutation is offered as a validated change.

// @Summary List redis connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/redis/connections [get]
func ListRedisConnections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		conns, err := d.connections(ctx, d.Redis)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a redis connection (database is the db index)
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections [post]
func SaveRedisConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return d.saveConnection(ctx, d.Redis, c)
	})
}

// @Summary Get one redis connection (password redacted)
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/redis/connections/{name} [get]
func GetRedisConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.connection(ctx, d.Redis, r.PathValue("name"))
	})
}

// @Summary Delete a redis connection
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name} [delete]
func DeleteRedisConnection(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		return d.deleteConnection(ctx, d.Redis, r.PathValue("name"))
	})
}

// @Summary Test that a redis connection answers
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name}/test [post]
func TestRedisConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if err := d.testConnection(ctx, d.Redis, r.PathValue("name")); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Success 200 {object} redis.KeyPage
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name}/keys [get]
func ListRedisKeys(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Redis, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return d.Redis.Keys(ctx, *c, r.URL.Query().Get("pattern"), limit)
	})
}

// @Summary Inspect one key's value (rendered per its type)
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name}/value [get]
func GetRedisValue(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		key := r.URL.Query().Get("key")
		if key == "" {
			return nil, fmt.Errorf("key is required")
		}
		c, err := d.raw(ctx, d.Redis, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Redis.Value(ctx, *c, key)
	})
}

type redisKeysBody struct {
	Keys []string `json:"keys"`
}

// @Summary Remove keys (the selected ones only)
// @Param name path string true "connection name"
// @Param body body redisKeysBody true "keys to remove"
// @Success 200 {object} object{deleted=int}
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name}/keys [delete]
func DeleteRedisKeys(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body redisKeysBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, d.Redis, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		deleted, err := d.Redis.Delete(ctx, *c, body.Keys)
		return map[string]any{"deleted": deleted}, err
	})
}

type redisFlushBody struct {
	Confirm string `json:"confirm"`
}

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param name path string true "connection name"
// @Param body body redisFlushBody true "confirm must repeat the connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name}/flush [post]
func FlushRedisDB(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body redisFlushBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, d.Redis, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		if body.Confirm != c.Name {
			return nil, fmt.Errorf("flushing %s needs its name typed as confirmation", c.Name)
		}
		if err := d.Redis.Flush(ctx, *c); err != nil {
			return nil, err
		}
		return map[string]string{"status": "flushed"}, nil
	})
}

// @Summary Redis INFO snapshot (memory, clients, hits, persistence)
// @Param name path string true "connection name"
// @Success 200 {object} redis.Stats
// @Failure 400 {object} map[string]string
// @Router /v1/redis/connections/{name}/stats [get]
func GetRedisStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Redis, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Redis.Stats(ctx, *c)
	})
}

// @Summary Redis stats stream (SSE, one INFO snapshot every 2s)
// @Param name path string true "connection name"
// @Success 200 {string} string "text/event-stream"
// @Router /v1/redis/connections/{name}/stats/stream [get]
func StreamRedisStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	streamSnapshots(w, r, func(ctx context.Context) (any, error) {
		c, err := d.raw(ctx, d.Redis, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Redis.Stats(ctx, *c)
	})
}
