package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"noxis/cmd/noxis/server/deps"
	"noxis/internal/entity"
)

// The memory-cache operations. Connection CRUD is shared generically with the
// other families; what is memory-cache-only is the key browser, the value
// inspector, key removal and emptying the database.
//
// Writes stop at removing keys: a cache value cannot be rolled back, so no key
// mutation is offered as a validated change.

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Success 200 {object} entity.KeyPage
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/keys [get]
func ListKeys(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		mc, err := d.Modules.EngineMemcache(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		c, err := d.Raw.Serve(ctx, mc, r.PathValue("name"), entity.LevelRead)
		if err != nil {
			return nil, err
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		return mc.Keys(ctx, *c, r.URL.Query().Get("pattern"), limit)
	})
}

// @Summary Inspect one key's value (rendered per its type)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/value [get]
func GetValue(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		key := r.URL.Query().Get("key")
		if key == "" {
			return nil, fmt.Errorf("key is required")
		}
		mc, err := d.Modules.EngineMemcache(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		c, err := d.Raw.Serve(ctx, mc, r.PathValue("name"), entity.LevelRead)
		if err != nil {
			return nil, err
		}
		return mc.Value(ctx, *c, key)
	})
}

type memcacheKeysBody struct {
	Keys []string `json:"keys"`
}

// @Summary Remove keys (the selected ones only)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body memcacheKeysBody true "keys to remove"
// @Success 200 {object} object{deleted=int}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/keys [delete]
func DeleteKeys(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		var body memcacheKeysBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		mc, err := d.Modules.EngineMemcache(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		c, err := d.Raw.Serve(ctx, mc, r.PathValue("name"), entity.LevelAdmin)
		if err != nil {
			return nil, err
		}
		deleted, err := mc.Delete(ctx, *c, body.Keys)
		return map[string]any{"deleted": deleted}, err
	})
}

type memcacheFlushBody struct {
	Confirm string `json:"confirm"`
}

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body memcacheFlushBody true "confirm must repeat the connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/flush [post]
func FlushDB(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		var body memcacheFlushBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		mc, err := d.Modules.EngineMemcache(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		c, err := d.Raw.Serve(ctx, mc, r.PathValue("name"), entity.LevelAdmin)
		if err != nil {
			return nil, err
		}
		if body.Confirm != c.Name {
			return nil, fmt.Errorf("flushing %s needs its name typed as confirmation", c.Name)
		}
		if err := mc.Flush(ctx, *c); err != nil {
			return nil, err
		}
		return entity.StatusResponse{Status: "flushed"}, nil
	})
}
