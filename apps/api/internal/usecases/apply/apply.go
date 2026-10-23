// Package apply holds the change-application use case and the scheduler pass.
// The lifecycle rules live with each family's change queue; these use cases
// authorize the caller, hand the change over, and answer.
package apply

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Apply applies an approved change.
type Apply struct{ usecases.RegistryDeps }

// @Summary Apply an approved change
// @Param engine path string true "engine family (sql, mongodb)"
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Failure 409 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes/{id}/apply [post]
func (uc Apply) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	ch, err := uc.changeFor(ctx, mod, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	// applying is the widest action a change can take: it needs admin on the
	// connection, not just the read grant that opened it
	if err := uc.Authorize(ctx, ch.Connection, entity.LevelAdmin); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	flow, err := uc.flowFor(ch)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := flow.Apply(ctx, ch, entity.Actor(ctx))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}

// flowFor is the change queue that owns this change's kind.
func (uc Apply) flowFor(ch *entity.Change) (interfaces.ChangeFlow, error) {
	flows := uc.Modules.Changes()
	if ch.Kind == constants.KindMongo {
		if flows.Mongo == nil {
			return nil, fmt.Errorf("%w: no change queue for %s", constants.ErrUnsupported, ch.Kind)
		}
		return flows.Mongo, nil
	}
	if flows.SQL == nil {
		return nil, fmt.Errorf("%w: no change queue for %s", constants.ErrUnsupported, ch.Kind)
	}
	return flows.SQL, nil
}

// changeFor loads one change for the addressed family after the family gate and
// the caller's read grant.
func (uc Apply) changeFor(ctx context.Context, mod interfaces.Module, id string) (*entity.Change, error) {
	switch mod.(type) {
	case interfaces.SQLModule:
		return uc.SQLChange(ctx, id)
	case interfaces.MongoModule:
		return uc.MongoChange(ctx, id)
	}
	return nil, fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, mod.Family())
}

// Tick applies every due scheduled change; the external scheduler drives it.
// Each family's queue runs its own due changes, so a pass is family-agnostic.
type Tick struct{ usecases.RegistryDeps }

// @Summary Apply every due scheduled change (external cron / scheduler)
// @Success 200 {object} entity.TickResult
// @Router /internal/callback/scheduler [post]
func (uc Tick) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	flows := uc.Modules.Changes()
	res := entity.TickResult{Applied: []string{}, Failed: []string{}}
	for _, flow := range []interfaces.ChangeFlow{flows.SQL, flows.Mongo} {
		if flow == nil {
			continue // a registry that serves no queue for this family
		}
		out, err := flow.Tick(ctx, time.Now().Unix(), constants.SchedulerActor)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		res.Applied = append(res.Applied, out.Applied...)
		res.Failed = append(res.Failed, out.Failed...)
	}
	status := "ok"
	if len(res.Failed) > 0 {
		status = "failed"
	}
	uc.Metrics.SchedulerTick(ctx, status)
	uc.Respond(w, res)
}
