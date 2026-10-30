package instance

import (
	"net/http"
	"strings"

	"noxis/internal/common/logging"
	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List lists instances across drivers (passwords redacted).
type List struct{ usecases.RegistryDeps }

// @Summary List instances across drivers (passwords redacted)
// @Param name query string false "substring match on name"
// @Param driver query string false "exact driver match"
// @Param host query string false "substring match on host"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} entity.InstanceList
// @Router /external/v1/instances [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	query := r.URL.Query()
	all, err := uc.Repositories.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	nameNeedle := strings.ToLower(query.Get("name"))
	driver := query.Get("driver")
	hostNeedle := strings.ToLower(query.Get("host"))
	out := make([]entity.Connection, 0, len(all))
	for i := range all {
		connection := &all[i]
		if _, err := uc.Modules.Engine(connection.Driver); err != nil {
			continue
		}
		if nameNeedle != "" && !strings.Contains(strings.ToLower(connection.Name), nameNeedle) {
			continue
		}
		if driver != "" && !strings.EqualFold(connection.Driver, driver) {
			continue
		}
		if hostNeedle != "" && !strings.Contains(strings.ToLower(connection.Host), hostNeedle) {
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
		out = append(out, connection.Redacted())
	}
	page, perPage := pagination.Params(query.Get("page"), query.Get("perPage"), constants.DefaultPageSize)
	uc.Respond(w, entity.InstanceList{Instances: pagination.Slice(out, page, perPage), Total: len(out)})
}
