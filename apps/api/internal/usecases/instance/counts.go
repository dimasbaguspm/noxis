package instance

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Counts counts the readable instances of each driver, so the console's driver list can show a number per engine
// without asking once per driver.
type Counts struct{ usecases.RegistryDeps }

// @Summary Count readable instances per driver
// @Success 200 {object} entity.InstanceCounts
// @Router /external/v1/instances/counts [get]
func (uc Counts) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	all, err := uc.Repositories.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	counts := make(map[string]int, len(all))
	for i := range all {
		connection := &all[i]
		// a driver with no module is not addressable, so its rows are not counted either
		if _, err := uc.Modules.Engine(connection.Driver); err != nil {
			continue
		}
		readable, err := uc.CanRead(ctx, connection.Name)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		if !readable {
			continue
		}
		counts[connection.Driver]++
	}
	uc.Respond(w, entity.InstanceCounts{Counts: counts})
}
