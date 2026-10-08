// Package server is the noxis REST API and operator console.
//
// @title       noxis API
// @version     0.1.0
// @description SQL console API: manage database connections and run
// verified reads, writes, dry-run rollbacks and analytics against them.
package server

import (
	"log/slog"
	"net/http"

	"noxis/cmd/noxis/server/controller"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/sqlite"
	"noxis/internal/services/memcache"
	"noxis/internal/services/mongodb"
	"noxis/internal/services/sql"
)

// newDeps wires the shared stores and the engine-family registry.
func newDeps(store *sqlite.Store, log *slog.Logger, tickKey string) controller.Deps {
	return controller.Deps{
		Connections: store,
		Changes:     store,
		Modules: map[string]interfaces.Module{
			constants.FamilySQL:      sql.New(),
			constants.FamilyMemcache: memcache.New(),
			constants.FamilyMongo:    mongodb.New(),
		},
		Log:     log,
		TickKey: tickKey,
	}
}

// newHandler wires routes to handlers; dependencies ride the request ctx.
//
// Routes are generic over the engine family: `{engine}` names the family and
// every handler resolves its module from the registry. A family is registered
// exactly once, here, and a family that does not serve a route is refused by
// the handler (404) instead of reaching the wrong module.
func newHandler(deps controller.Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", controller.GetHealthz)
	mux.HandleFunc("GET /v1/connections/status/stream", controller.StreamConnectionStatus)

	// Connections: one generic handler per route, gated by the {engine} path
	// parameter through the module registry.
	mux.HandleFunc("GET /v1/{engine}/connections", controller.ListConnections)
	mux.HandleFunc("POST /v1/{engine}/connections", controller.SaveConnection)
	mux.HandleFunc("GET /v1/{engine}/connections/{name}", controller.GetConnection)
	mux.HandleFunc("DELETE /v1/{engine}/connections/{name}", controller.DeleteConnection)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/test", controller.TestConnection)

	// SQL-only: schema, row preview and the console ops.
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/schema", controller.GetSchema)
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/tables/{table}/rows", controller.GetRows)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/query", controller.PostQuery)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/alter", controller.PostAlter)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/rollback", controller.PostRollback)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/analytics", controller.PostAnalytics)

	// Stats: one handler, the engine's capability decides the shape.
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/stats", controller.GetStats)
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/stats/stream", controller.StreamStats)

	// Changes: SQL and MongoDB keep separate queues behind one route.
	mux.HandleFunc("GET /v1/{engine}/changes", controller.ListChanges)
	mux.HandleFunc("POST /v1/{engine}/changes", controller.SubmitChange)
	mux.HandleFunc("GET /v1/{engine}/changes/{id}", controller.GetChange)
	mux.HandleFunc("POST /v1/{engine}/changes/{id}/approve", controller.ApproveChange)
	mux.HandleFunc("POST /v1/{engine}/changes/{id}/apply", controller.ApplyChange)
	mux.HandleFunc("POST /v1/{engine}/changes/{id}/reject", controller.RejectChange)

	// Memory-cache-only: keys, values, removal and emptying the database.
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/keys", controller.ListKeys)
	mux.HandleFunc("DELETE /v1/{engine}/connections/{name}/keys", controller.DeleteKeys)
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/value", controller.GetValue)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/flush", controller.FlushDB)

	// MongoDB-only: databases, collections and read requests.
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/databases", controller.ListDatabases)
	mux.HandleFunc("GET /v1/{engine}/connections/{name}/collections", controller.ListCollections)
	mux.HandleFunc("POST /v1/{engine}/connections/{name}/requests", controller.PostRequest)

	mux.Handle("GET /assets/", http.HandlerFunc(controller.GetAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", controller.GetConsole)                     // the operator console (SPA shell)

	// Scheduling: an external cron (or the in-process scheduler) drives due
	// changes through the same handler.
	mux.HandleFunc("POST /internal/tick", controller.PostTick)

	return withMiddleware(controller.WithDeps(mux, deps), deps.Log)
}
