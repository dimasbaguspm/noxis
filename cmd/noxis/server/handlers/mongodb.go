package handlers

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
	"noxis/internal/usecases"
)

// The MongoDB operations. Connection CRUD and the change routes are shared
// generically with the other families; what is MongoDB-only is the
// database/collection explorer and the read request runner.
//
// Requests run directly: a document change has no generic rollback, so the
// guards are the op allowlist and the refusal to update or delete without a
// filter (an empty filter would touch every document).

// @Summary List databases with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{databases=[]entity.Database}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/databases [get]
func ListDatabases(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		m, err := d.Engines.EngineMongo(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		c, err := d.Connections.Raw(ctx, m, r.PathValue("name"), entity.LevelRead)
		if err != nil {
			return nil, err
		}
		databases, err := m.Databases(ctx, *c)
		return map[string]any{"databases": databases}, err
	})
}

// @Summary List a database's collections with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} object{collections=[]entity.Collection}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/collections [get]
func ListCollections(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		m, err := d.Engines.EngineMongo(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		c, err := d.Connections.Raw(ctx, m, r.PathValue("name"), entity.LevelRead)
		if err != nil {
			return nil, err
		}
		collections, err := m.Collections(ctx, *c, r.URL.Query().Get("database"))
		return map[string]any{"collections": collections}, err
	})
}

// @Summary Run a read request (find | aggregate | count)
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param body body entity.MongoRequest true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /v1/engine/{engine}/connections/{name}/requests [post]
func PostRequest(w http.ResponseWriter, r *http.Request) {
	d := usecases.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		var req interfaces.MongoRequest
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		m, err := d.Engines.EngineMongo(r.PathValue("engine"))
		if err != nil {
			return nil, err
		}
		// a write op is refused by Run, but it must still not be handed to a
		// read-only caller
		level := entity.LevelRead
		if mongodb.IsWrite(req.Op) {
			level = entity.LevelWrite
		}
		c, err := d.Connections.Raw(ctx, m, r.PathValue("name"), level)
		if err != nil {
			return nil, err
		}
		return m.Run(ctx, *c, req)
	})
}

// mongoChangeBody is one submitted request: the connection and notes of a
// change, plus the request itself, flat so the console sends one object.
type mongoChangeBody struct {
	Connection string `json:"connection"`
	Notes      string `json:"notes"`
	interfaces.MongoRequest
}
