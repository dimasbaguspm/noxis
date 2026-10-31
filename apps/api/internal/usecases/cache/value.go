package cache

import (
	"context"
	"errors"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Value inspects one key's value.
type Value struct{ usecases.RegistryDeps }

// @Summary Inspect one key's value (rendered per its type)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/value [get]
func (uc Value) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.KeyBrowser](uc.RegistryDeps, w, r, "Value", entity.LevelRead,
		func(ctx context.Context, mc interfaces.KeyBrowser, conn entity.Connection) (any, error) {
			key := r.URL.Query().Get("key")
			if key == "" {
				return nil, errors.New("key is required")
			}
			return mc.Value(ctx, conn, key)
		})
}
