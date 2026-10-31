package cache

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// ttlRequest names the keys and the expiry to give them: zero clears it.
type ttlRequest struct {
	Keys  []string `json:"keys"`
	TTLMs int64    `json:"ttlMs"`
}

// TTL sets or clears the expiry of the named keys. It is the one mutation this family offers besides delete, because a
// TTL is metadata and reversible while a value is neither.
type TTL struct{ usecases.RegistryDeps }

// @Summary Set or clear the expiry of the named keys (ttlMs 0 clears it)
// @Param driver path string true "driver name (redis, valkey, dragonfly)"
// @Param name path string true "connection name"
// @Param body body ttlRequest true "keys and ttlMs"
// @Success 200 {object} map[string]int64
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/keys/ttl [post]
func (uc TTL) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body ttlRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var mc interfaces.KeyBrowser
	if err := uc.RouteModule(ctx, r, &mc); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mc, r.PathValue("name"), entity.LevelWrite)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	changed, err := mc.Expire(ctx, *c, body.Keys, body.TTLMs)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, map[string]int64{"changed": changed})
}
