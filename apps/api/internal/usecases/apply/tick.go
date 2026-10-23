package apply

import (
	"net/http"
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
	flows := uc.Modules.Changes()
	result := entity.TickResult{Applied: []string{}, Failed: []string{}}
	for _, flow := range []interfaces.ChangeFlow{flows.SQL, flows.Mongo} {
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
