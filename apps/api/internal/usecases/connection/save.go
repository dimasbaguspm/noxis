package connection

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Save creates or updates a connection.
type Save struct{ usecases.RegistryDeps }

// @Summary Create or update a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections [post]
func (uc Save) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var connection entity.Connection
	if err := uc.Decode(r, &connection); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if !module.Supports(connection.Driver) {
		uc.Fail(ctx, w, fmt.Errorf("connection driver %q does not match the engine path %q", connection.Driver, module.Driver()))
		return
	}
	if err := connection.Validate(); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	existing, err := uc.Repositories.Connections.Get(ctx, connection.Name)
	switch {
	case err == nil:
		if connection.Driver != existing.Driver || connection.Host != existing.Host ||
			connection.Database != existing.Database || connection.User != existing.User {
			uc.Fail(ctx, w, errors.New(
				"driver, host, database and user cannot be changed after registration; "+
					"rotate the password or register a new connection"))
			return
		}
		if connection.Password == "" {
			connection.Password = existing.Password
		}
		connection.CreatedAt = existing.CreatedAt
		connection.CreatedBy = existing.CreatedBy
		connection.ID = existing.ID
	case errors.Is(err, constants.ErrNotFound):
		connection.CreatedAt = time.Now().Unix()
		connection.CreatedBy = entity.Actor(ctx)
		connection.ID = idgen.UUID()
	default:
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Repositories.Connections.Save(ctx, connection); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, connection.Redacted())
}
