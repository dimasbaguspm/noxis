package mongo

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"noxis/internal/usecases"
)

// Run runs a read request (find | aggregate | count).
type Run struct{ usecases.RegistryDeps }

// @Summary Run a read request (find | aggregate | count)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param body body entity.MongoRequest true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/requests [post]
func (uc Run) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var request interfaces.MongoRequest
	if err := uc.Decode(r, &request); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var module interfaces.DocumentBrowser
	if err := uc.RouteModule(ctx, r, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	level := entity.LevelRead
	if mongodb.IsWrite(request.Op) {
		level = entity.LevelWrite
	}
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), level)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	result, err := module.Run(ctx, *connection, request)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, result)
}
