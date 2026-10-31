package cache

import (
	"context"
	"fmt"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Flush empties the connected database (FLUSHDB, never FLUSHALL).
type Flush struct{ usecases.RegistryDeps }

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param body body entity.CacheFlushRequest true "confirm must repeat the connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/flush [post]
func (uc Flush) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.KeyBrowser](uc.RegistryDeps, w, r, "Flush", entity.LevelAdmin,
		func(ctx context.Context, mc interfaces.KeyBrowser, conn entity.Connection) (any, error) {
			var body entity.CacheFlushRequest
			if err := uc.Decode(r, &body); err != nil {
				return nil, err
			}
			if body.Confirm != conn.Name {
				return nil, fmt.Errorf("flushing %s needs its name typed as confirmation", conn.Name)
			}
			if err := mc.Flush(ctx, conn); err != nil {
				return nil, err
			}
			return entity.StatusResponse{Status: "flushed"}, nil
		})
}
