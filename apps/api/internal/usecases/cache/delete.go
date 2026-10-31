package cache

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Delete removes the selected keys only.
type Delete struct{ usecases.RegistryDeps }

// @Summary Remove keys (the selected ones only)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param body body entity.CacheDeleteRequest true "keys to remove"
// @Success 200 {object} entity.CacheDeleteResult
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/keys [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.KeyBrowser](uc.RegistryDeps, w, r, "Delete", entity.LevelAdmin,
		func(ctx context.Context, mc interfaces.KeyBrowser, conn entity.Connection) (any, error) {
			var body entity.CacheDeleteRequest
			if err := uc.Decode(r, &body); err != nil {
				return nil, err
			}
			deleted, err := mc.Delete(ctx, conn, body.Keys)
			if err != nil {
				return nil, err
			}
			return entity.CacheDeleteResult{Deleted: deleted}, nil
		})
}
