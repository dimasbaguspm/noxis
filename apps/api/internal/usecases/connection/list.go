package connection

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List lists the caller's readable connections (passwords redacted).
type List struct{ usecases.RegistryDeps }

// @Summary List connections (passwords redacted)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Success 200 {object} entity.ConnectionList
// @Router /external/v1/engine/{driver}/connections [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	connections, err := uc.Repositories.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out := make([]entity.Connection, 0, len(connections))
	for i := range connections {
		connection := &connections[i]
		if !module.Supports(connection.Driver) {
			continue
		}
		readable, err := uc.CanRead(ctx, connection.Name)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		if readable {
			out = append(out, connection.Redacted())
		}
	}
	uc.Respond(w, entity.ConnectionList{Connections: out})
}
