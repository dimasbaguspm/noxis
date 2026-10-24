package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Schedule schedules or unschedules an approved change.
type Schedule struct{ usecases.RegistryDeps }

// @Summary Schedule or unschedule an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Param body body entity.ScheduleRequest true "scheduledAt unix seconds; 0 clears the schedule"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/schedule [post]
func (uc Schedule) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body entity.ScheduleRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	decide(ctx, uc.RegistryDeps, w, r, func(change *entity.Change) error {
		actor := entity.Actor(ctx)
		if change.Kind == constants.KindMongo {
			return uc.Modules.Changes().Mongo.Schedule(ctx, change, body.ScheduledAt, actor)
		}
		return uc.Modules.Changes().SQL.Schedule(ctx, change, body.ScheduledAt, actor)
	})
}
