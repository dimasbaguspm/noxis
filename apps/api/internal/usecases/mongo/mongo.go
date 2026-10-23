// Package mongo holds the MongoDB explorer use cases. Requests run directly:
// a document change has no generic rollback, so the guards are the op
// allowlist and the refusal to update or delete without a filter.
package mongo

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"noxis/internal/usecases"
)

// Databases lists databases with their sizes.
type Databases struct{ usecases.RegistryDeps }

// @Summary List databases with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.DatabaseList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/databases [get]
func (uc Databases) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	m, err := uc.Modules.EngineMongo(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, m, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := m.Databases(ctx, *c)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.DatabaseList{Databases: out})
}

// Collections lists a database's collections with their sizes.
type Collections struct{ usecases.RegistryDeps }

// @Summary List a database's collections with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} entity.CollectionList
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/collections [get]
func (uc Collections) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	m, err := uc.Modules.EngineMongo(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, m, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := m.Collections(ctx, *c, r.URL.Query().Get("database"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.CollectionList{Collections: out})
}

// Run runs a read request (find | aggregate | count).
type Run struct{ usecases.RegistryDeps }

// @Summary Run a read request (find | aggregate | count)
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param body body entity.MongoRequest true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/requests [post]
func (uc Run) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var req interfaces.MongoRequest
	if err := uc.Decode(r, &req); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	m, err := uc.Modules.EngineMongo(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	// a write op is refused by Run, but it must still not be handed to a
	// read-only caller
	level := entity.LevelRead
	if mongodb.IsWrite(req.Op) {
		level = entity.LevelWrite
	}
	c, err := uc.RawConn(ctx, m, r.PathValue("name"), level)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := m.Run(ctx, *c, req)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}
