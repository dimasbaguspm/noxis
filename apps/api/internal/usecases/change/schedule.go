package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Schedule schedules or unschedules an approved change.
type Schedule struct{ usecases.RegistryDeps }

func (uc Schedule) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var body entity.ScheduleRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	decide(ctx, uc.RegistryDeps, w, r, func(change *entity.Change) error {
		flow, err := flowFor(ctx, change)
		if err != nil {
			return err
		}
		return flow.Schedule(ctx, change, body.ScheduledAt, entity.Actor(ctx))
	})
}
