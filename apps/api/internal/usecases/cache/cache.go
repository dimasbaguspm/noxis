// Package cache holds the memory-cache use cases. A cache value cannot be
// rolled back, so no key mutation is offered as a validated change; writes
// stop at removing keys.
package cache

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Keys browses keys by pattern (bounded scan).
type Keys struct{ usecases.RegistryDeps }

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Success 200 {object} entity.KeyPage
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/keys [get]
func (uc Keys) Serve(w http.ResponseWriter, r *http.Request) {
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	c, err := uc.RawConn(r.Context(), mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := mc.Keys(r.Context(), *c, r.URL.Query().Get("pattern"), limit)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// Value inspects one key's value.
type Value struct{ usecases.RegistryDeps }

// @Summary Inspect one key's value (rendered per its type)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/value [get]
func (uc Value) Serve(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		uc.Fail(w, errors.New("key is required"))
		return
	}
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	c, err := uc.RawConn(r.Context(), mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	out, err := mc.Value(r.Context(), *c, key)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, out)
}

// keysBody is the remove-keys payload.
type keysBody struct {
	Keys []string `json:"keys"`
}

// Delete removes the selected keys only.
type Delete struct{ usecases.RegistryDeps }

// @Summary Remove keys (the selected ones only)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body keysBody true "keys to remove"
// @Success 200 {object} object{deleted=int}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/keys [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	var body keysBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	c, err := uc.RawConn(r.Context(), mc, r.PathValue("name"), entity.LevelAdmin)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	deleted, err := mc.Delete(r.Context(), *c, body.Keys)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, map[string]any{"deleted": deleted})
}

// flushBody is the flush payload: confirm must repeat the connection name.
type flushBody struct {
	Confirm string `json:"confirm"`
}

// Flush empties the connected database (FLUSHDB, never FLUSHALL).
type Flush struct{ usecases.RegistryDeps }

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body flushBody true "confirm must repeat the connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/flush [post]
func (uc Flush) Serve(w http.ResponseWriter, r *http.Request) {
	var body flushBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	c, err := uc.RawConn(r.Context(), mc, r.PathValue("name"), entity.LevelAdmin)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if body.Confirm != c.Name {
		uc.Fail(w, fmt.Errorf("flushing %s needs its name typed as confirmation", c.Name))
		return
	}
	if err := mc.Flush(r.Context(), *c); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, entity.StatusResponse{Status: "flushed"})
}
