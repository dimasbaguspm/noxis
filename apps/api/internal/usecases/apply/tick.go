package apply

import (
	"net/http"
	"sort"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
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
	var flows interfaces.ChangeFlows
	if err := uc.RouteModule(ctx, r, &flows); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	result := entity.TickResult{Applied: []string{}, Failed: []string{}}
	drivers := make([]string, 0, len(flows))
	for driver := range flows {
		drivers = append(drivers, driver)
	}
	sort.Strings(drivers)
	for _, driver := range drivers {
		flow := flows[driver]
		if flow == nil {
			continue
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
