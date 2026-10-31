package cache

import (
	"context"
	"net/http"

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
	usecases.RunOp[interfaces.KeyBrowser](uc.RegistryDeps, w, r, "TTL", entity.LevelWrite,
		func(ctx context.Context, mc interfaces.KeyBrowser, conn entity.Connection) (any, error) {
			var body ttlRequest
			if err := uc.Decode(r, &body); err != nil {
				return nil, err
			}
			changed, err := mc.Expire(ctx, conn, body.Keys, body.TTLMs)
			if err != nil {
				return nil, err
			}
			return map[string]int64{"changed": changed}, nil
		})
}
