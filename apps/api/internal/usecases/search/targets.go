package search

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Targets lists what a search platform indexes, so a console can pick a target before it queries one.
type Targets struct{ usecases.RegistryDeps }

// @Summary List the indices a search platform holds
// @Param driver path string true "driver name (elasticsearch)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.SearchTargetList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/targets [get]
func (uc Targets) Serve(w http.ResponseWriter, r *http.Request) {
	usecases.RunOp[interfaces.SearchEngine](uc.RegistryDeps, w, r, "Targets", entity.LevelRead,
		func(ctx context.Context, module interfaces.SearchEngine, conn entity.Connection) (any, error) {
			targets, err := module.Targets(ctx, conn)
			return entity.SearchTargetList{Targets: targets}, err
		})
}
