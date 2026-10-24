package schema

import (
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

var opNames = map[string]bool{
	constants.OpQuery:    true,
	constants.OpAlter:    true,
	constants.OpRollback: true,
	constants.OpAnalyze:  true,
}

// Op runs a direct SQL console operation.
type Op struct{ usecases.RegistryDeps }

// @Summary Run a SQL console operation
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body entity.OpRequest true "op plus sql or table"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/operations [post]
func (uc Op) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	ctx = common.WithModuleFamily(ctx, r.PathValue("engine"))
	var module interfaces.SQLModule
	if err := common.LoadModule(ctx, &module); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var body entity.OpRequest
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	operation := body.Op
	if !opNames[operation] {
		uc.Fail(ctx, w, fmt.Errorf("unknown operation %q", operation))
		return
	}
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
