package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Approve approves a validated change (self-approval allowed).
type Approve struct{ usecases.RegistryDeps }

// @Summary Approve a validated change (self-approval allowed)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/changes/{id}/approve [post]
func (uc Approve) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	decide(ctx, uc.RegistryDeps, w, r, func(change *entity.Change) error {
		return uc.ApproveChange(ctx, change, entity.Actor(ctx))
	})
}
