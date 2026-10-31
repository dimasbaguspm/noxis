package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Reject rejects a change and keeps it as an artifact.
type Reject struct{ usecases.RegistryDeps }

// @Summary Reject a change (kept as an artifact, never deleted)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/changes/{id}/reject [post]
func (uc Reject) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	decide(ctx, uc.RegistryDeps, w, r, func(change *entity.Change) error {
		return uc.RejectChange(ctx, change, entity.Actor(ctx))
	})
}
