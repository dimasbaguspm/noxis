package controller

import (
	"context"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The MongoDB namespace. Connection CRUD shares one implementation with the
// other families; what is MongoDB-only is the database/collection explorer,
// the CRUD request runner and the stats.
//
// Requests run directly: a document change has no generic rollback, so the
// guards are the op allowlist and the refusal to update or delete without a
// filter (an empty filter would touch every document).

// @Summary List MongoDB connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/mongodb/connections [get]
func ListMongoConnections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		conns, err := d.connections(ctx, d.Mongo)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a MongoDB connection
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections [post]
func SaveMongoConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return d.saveConnection(ctx, d.Mongo, c)
	})
}

// @Summary Get one MongoDB connection (password redacted)
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/mongodb/connections/{name} [get]
func GetMongoConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.connection(ctx, d.Mongo, r.PathValue("name"))
	})
}

// @Summary Delete a MongoDB connection
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name} [delete]
func DeleteMongoConnection(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		return d.deleteConnection(ctx, d.Mongo, r.PathValue("name"))
	})
}

// @Summary Test that a MongoDB connection answers
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/test [post]
func TestMongoConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if err := d.testConnection(ctx, d.Mongo, r.PathValue("name")); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary List databases with their sizes
// @Param name path string true "connection name"
// @Success 200 {object} object{databases=[]mongodb.Database}
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/databases [get]
func ListMongoDatabases(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		databases, err := d.Mongo.Databases(ctx, *c)
		return map[string]any{"databases": databases}, err
	})
}

// @Summary List a database's collections with their sizes
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} object{collections=[]mongodb.Collection}
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/collections [get]
func ListMongoCollections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		collections, err := d.Mongo.Collections(ctx, *c, r.URL.Query().Get("database"))
		return map[string]any{"collections": collections}, err
	})
}

// @Summary Run a CRUD request (find | aggregate | count | insert | update | delete)
// @Param name path string true "connection name"
// @Param body body mongodb.Request true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} entity.Result
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/requests [post]
func PostMongoRequest(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var req interfaces.MongoRequest
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Mongo.Run(ctx, *c, req)
	})
}

// @Summary MongoDB stats snapshot (server, database and collection)
// @Param name path string true "connection name"
// @Param database query string false "database to describe"
// @Param collection query string false "collection to describe"
// @Success 200 {object} mongodb.Stats
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/stats [get]
func GetMongoStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.mongoStats(ctx, r)
	})
}

// @Summary MongoDB stats stream (SSE, one snapshot every 2s)
// @Param name path string true "connection name"
// @Param database query string false "database to describe"
// @Param collection query string false "collection to describe"
// @Success 200 {string} string "text/event-stream"
// @Router /v1/mongodb/connections/{name}/stats/stream [get]
func StreamMongoStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	streamSnapshots(w, r, func(ctx context.Context) (any, error) { return d.mongoStats(ctx, r) })
}

func (d Deps) mongoStats(ctx context.Context, r *http.Request) (out any, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	return d.Mongo.Stats(ctx, *c, r.URL.Query().Get("database"), r.URL.Query().Get("collection"))
}
