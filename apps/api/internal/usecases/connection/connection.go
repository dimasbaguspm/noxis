// Package connection holds the connection use cases.
package connection

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List lists the caller's readable connections (passwords redacted).
type List struct{ usecases.RegistryDeps }

// @Summary List connections (passwords redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Success 200 {object} entity.ConnectionList
// @Router /external/v1/engine/{engine}/connections [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	list, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out := make([]entity.Connection, 0, len(list))
	for i := range list {
		c := &list[i]
		if !mod.Supports(c.Driver) {
			continue
		}
		ok, err := uc.CanRead(ctx, c.Name)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		if ok {
			out = append(out, c.Redacted())
		}
	}
	uc.Respond(w, entity.ConnectionList{Connections: out})
}

// Get returns one connection (password redacted).
type Get struct{ usecases.RegistryDeps }

// @Summary Get one connection (password redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mod, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, c.Redacted())
}

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
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var c entity.Connection
	if err := uc.Decode(r, &c); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if !mod.Supports(c.Driver) {
		uc.Fail(ctx, w, fmt.Errorf("driver %q is not a %s driver", c.Driver, mod.Family()))
		return
	}
	if err := c.Validate(); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	existing, err := uc.Connections.Get(ctx, c.Name)
	switch {
	case err == nil:
		// identity is fixed at registration: a different driver, host,
		// database or user is a different instance, not an edit. The password
		// rotates through its own action.
		if c.Driver != existing.Driver || c.Host != existing.Host ||
			c.Database != existing.Database || c.User != existing.User {
			uc.Fail(ctx, w, errors.New(
				"driver, host, database and user cannot be changed after registration; "+
					"rotate the password or register a new connection"))
			return
		}
		if c.Password == "" {
			c.Password = existing.Password
		}
		c.CreatedAt = existing.CreatedAt
		c.CreatedBy = existing.CreatedBy
		c.ID = existing.ID
	case errors.Is(err, constants.ErrNotFound):
		c.CreatedAt = time.Now().Unix()
		c.CreatedBy = entity.Actor(ctx)
		c.ID = idgen.UUID()
	default:
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Connections.Save(ctx, c); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, c.Redacted())
}

// Delete removes a connection and its changes.
type Delete struct{ usecases.RegistryDeps }

// @Summary Delete a connection (and its changes)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name} [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, err := uc.RawConn(ctx, mod, r.PathValue("name"), entity.LevelRead); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Connections.Delete(ctx, r.PathValue("name")); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}

// Test pings a registered connection.
type Test struct{ usecases.RegistryDeps }

// @Summary Test that a connection answers
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/test [post]
func (uc Test) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mod, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := mod.Ping(ctx, *c); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.StatusResponse{Status: "ok"})
}
