package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"noxis/internal/entity"
)

// The Memcache namespace. Connection CRUD shares one implementation with SQL
// (the module decides which drivers belong to it); what is Memcache-only is the
// key browser, the value inspector, key removal and INFO stats.
//
// Writes stop at removing keys: a Memcache value cannot be rolled back, so no
// key mutation is offered as a validated change.

// @Summary List redis connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/memcache/connections [get]
func ListMemcacheConnections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		conns, err := d.connections(ctx, d.Memcache)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a redis connection (database is the db index)
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections [post]
func SaveMemcacheConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return d.saveConnection(ctx, d.Memcache, c)
	})
}

// @Summary Get one redis connection (password redacted)
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/memcache/connections/{name} [get]
func GetMemcacheConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.connection(ctx, d.Memcache, r.PathValue("name"))
	})
}

// @Summary Delete a redis connection
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name} [delete]
func DeleteMemcacheConnection(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		return d.deleteConnection(ctx, d.Memcache, r.PathValue("name"))
	})
}

// @Summary Test that a redis connection answers
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name}/test [post]
func TestMemcacheConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if err := d.testConnection(ctx, d.Memcache, r.PathValue("name")); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Success 200 {object} memcache.KeyPage
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name}/keys [get]
func ListMemcacheKeys(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Memcache, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return d.Memcache.Keys(ctx, *c, r.URL.Query().Get("pattern"), limit)
	})
}

// @Summary Inspect one key's value (rendered per its type)
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name}/value [get]
func GetMemcacheValue(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		key := r.URL.Query().Get("key")
		if key == "" {
			return nil, fmt.Errorf("key is required")
		}
		c, err := d.raw(ctx, d.Memcache, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Memcache.Value(ctx, *c, key)
	})
}

type memcacheKeysBody struct {
	Keys []string `json:"keys"`
}

// @Summary Remove keys (the selected ones only)
// @Param name path string true "connection name"
// @Param body body memcacheKeysBody true "keys to remove"
// @Success 200 {object} object{deleted=int}
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name}/keys [delete]
func DeleteMemcacheKeys(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body memcacheKeysBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, d.Memcache, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		deleted, err := d.Memcache.Delete(ctx, *c, body.Keys)
		return map[string]any{"deleted": deleted}, err
	})
}

type memcacheFlushBody struct {
	Confirm string `json:"confirm"`
}

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param name path string true "connection name"
// @Param body body memcacheFlushBody true "confirm must repeat the connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name}/flush [post]
func FlushMemcacheDB(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body memcacheFlushBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, d.Memcache, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		if body.Confirm != c.Name {
			return nil, fmt.Errorf("flushing %s needs its name typed as confirmation", c.Name)
		}
		if err := d.Memcache.Flush(ctx, *c); err != nil {
			return nil, err
		}
		return map[string]string{"status": "flushed"}, nil
	})
}

// @Summary Memcache INFO snapshot (memory, clients, hits, persistence)
// @Param name path string true "connection name"
// @Success 200 {object} memcache.Stats
// @Failure 400 {object} map[string]string
// @Router /v1/memcache/connections/{name}/stats [get]
func GetMemcacheStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Memcache, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Memcache.Stats(ctx, *c)
	})
}

// @Summary Memcache stats stream (SSE, one INFO snapshot every 2s)
// @Param name path string true "connection name"
// @Success 200 {string} string "text/event-stream"
// @Router /v1/memcache/connections/{name}/stats/stream [get]
func StreamMemcacheStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	streamSnapshots(w, r, func(ctx context.Context) (any, error) {
		c, err := d.raw(ctx, d.Memcache, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Memcache.Stats(ctx, *c)
	})
}
