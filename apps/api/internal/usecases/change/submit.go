package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Submit submits a change after validating it.
type Submit struct{ usecases.RegistryDeps }

// @Summary Submit a change (validates before recording)
// @Param engine path string true "engine family (sql, mongodb)"
// @Param body body entity.Change true "connection, notes, query, rollback (sql) or the request (mongodb)"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/changes [post]
func (uc Submit) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	switch engine := module.(type) {
	case interfaces.SQLModule:
		var change entity.Change
		if err := uc.Decode(r, &change); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		connection, err := uc.RawConn(ctx, engine, change.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		var flow interfaces.SQLChangeFlow
		if err := uc.RouteModule(ctx, r, &flow); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := flow.Submit(ctx, *connection, entity.Actor(ctx), change)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, module.Driver())
		uc.Respond(w, out)
	case interfaces.MongoModule:
		var body entity.MongoChangeRequest
		if err := uc.Decode(r, &body); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		connection, err := uc.RawConn(ctx, engine, body.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		var flow interfaces.MongoChangeFlow
		if err := uc.RouteModule(ctx, r, &flow); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := flow.Submit(ctx, *connection, entity.Actor(ctx), body.Notes, body.MongoRequest)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, module.Driver())
		uc.Respond(w, out)
	default:
		uc.Fail(ctx, w, unsupportedChangeFlow(module))
	}
}

// SwaggerMongoChangeBody emits the MongoDB submit body schema.
//
// @Param body body entity.MongoChangeRequest true "mongodb request"
func SwaggerMongoChangeBody() {}
