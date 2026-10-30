package apply

import (
	"errors"
	"net/http"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Tick applies every due scheduled change; the external scheduler drives it.
type Tick struct{ usecases.RegistryDeps }

// @Summary Apply every due scheduled change (external cron / scheduler)
// @Success 200 {object} entity.TickResult
// @Router /internal/callback/scheduler [post]
func (uc Tick) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	result := entity.TickResult{Applied: []string{}, Failed: []string{}}
	// Every registered driver is asked for its own queue, so a new driver is scheduled without editing this loop.
	for _, driver := range uc.Modules.Drivers() {
		flow, err := uc.Modules.Changes(driver)
		if errors.Is(err, constants.ErrUnsupported) {
			continue // a driver with no change queue has nothing due
		}
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := flow.Tick(ctx, time.Now().Unix(), constants.SchedulerActor)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		result.Applied = append(result.Applied, out.Applied...)
		result.Failed = append(result.Failed, out.Failed...)
	}
	status := "ok"
	if len(result.Failed) > 0 {
		status = "failed"
	}
	uc.Metrics.SchedulerTick(ctx, status)
	uc.Respond(w, result)
}
