package connection

import (
	"fmt"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Probe pings a connection draft without saving it, so a form can prove a connection before it exists and the store
// never holds a row nobody verified.
type Probe struct{ usecases.RegistryDeps }

// @Summary Probe a connection draft without saving it
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param body body entity.Connection true "connection draft"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/probe [post]
func (uc Probe) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var draft entity.Connection
	if err := uc.Decode(r, &draft); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	// the probe opens a socket to a host the caller names, so it is the same gate as the save it precedes
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if !module.Supports(draft.Driver) {
		uc.Fail(ctx, w, fmt.Errorf("connection driver %q does not match the engine path %q", draft.Driver, module.Driver()))
		return
	}
	if err := draft.Validate(); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	// the draft carries the password the caller typed; nothing reads the store, and nothing writes it
	if err := module.Ping(ctx, draft); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.StatusResponse{Status: "ok"})
}
