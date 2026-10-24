package schema

import (
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

var opRoutes = map[string]string{
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/query":    constants.OpQuery,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/alter":    constants.OpAlter,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/rollback": constants.OpRollback,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/analyze":  constants.OpAnalyze,
}

// Op runs a direct SQL console operation.
type Op struct{ usecases.RegistryDeps }

// @Summary Run a SQL console op (query)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body entity.OpRequest true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/query [post]
func (uc Op) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var body entity.OpRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	operation := opRoutes[r.Pattern]
	connection, err := uc.RawConn(ctx, module, r.PathValue("name"), uc.OpLevel(operation))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	start := time.Now()
	var result any
	switch operation {
	case constants.OpQuery:
		result, err = module.Query(ctx, *connection, body.SQL)
	case constants.OpAlter:
		result, err = module.Alter(ctx, *connection, body.SQL)
	case constants.OpRollback:
		result, err = module.DryRun(ctx, *connection, body.SQL)
	case constants.OpAnalyze:
		result, err = module.Analyze(ctx, *connection, body.Table)
	default:
		err = fmt.Errorf("unknown op %q", operation)
	}
	uc.Metrics.OpDuration(ctx, r.PathValue("engine"), operation, time.Since(start).Seconds())
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, result)
}

// OpAlter documents the alter route.
// @Summary Apply a write directly (DML or DDL)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body entity.OpRequest true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/alter [post]
func OpAlter() {}

// OpRollback documents the rollback route.
// @Summary Dry-run statements inside a transaction that always rolls back
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body entity.OpRequest true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/rollback [post]
func OpRollback() {}

// OpAnalyze documents the analyze route.
// @Summary Refresh planner statistics for a table
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body entity.OpRequest true "table"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/analyze [post]
func OpAnalyze() {}
