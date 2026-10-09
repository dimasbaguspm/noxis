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
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/sqlite"
	"noxis/internal/services/memcache"
	"noxis/internal/services/mongodb"
	"noxis/internal/services/sql"
)

// newDeps wires the shared stores and the engine-family registry.
func newDeps(store *sqlite.Store, log *slog.Logger, cfg entity.Config) controller.Deps {
	return controller.Deps{
		Connections: store,
		Changes:     store,
		Grants:      store,
		Modules: map[string]interfaces.Module{
			constants.FamilySQL:      sql.New(),
			constants.FamilyMemcache: memcache.New(),
			constants.FamilyMongo:    mongodb.New(),
		},
		Log:     log,
		TickKey: cfg.TickKey,
		Auth: controller.AuthInfo{
			Enabled:  cfg.OIDCIssuer != "",
			Issuer:   cfg.OIDCIssuer,
			ClientID: cfg.OIDCClientID,
		},
	}
}

// newHandler wires routes to handlers; dependencies ride the request ctx.
//
// Routes are generic over the engine family: `{engine}` names the family and
// every handler resolves its module from the registry. A family is registered
// exactly once, here, and a family that does not serve a route is refused by
// the handler (404) instead of reaching the wrong module.
func newHandler(deps controller.Deps, verifier interfaces.Verifier) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", controller.GetHealthz)
	mux.HandleFunc("GET /v1/config/drivers", controller.ListDrivers)
	mux.HandleFunc("GET /auth/config", controller.GetAuthConfig)
	mux.HandleFunc("GET /v1/me", controller.GetMe)

	// Connections: one generic handler per route, gated by the {engine} path
	// parameter through the module registry.
	mux.HandleFunc("GET /v1/engine/{engine}/connections", controller.ListConnections)
	mux.HandleFunc("POST /v1/engine/{engine}/connections", controller.SaveConnection)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}", controller.GetConnection)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}", controller.DeleteConnection)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/test", controller.TestConnection)

	// Grants: per-connection RBAC policy, managed by global admins only.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/grants", controller.ListGrants)
	mux.HandleFunc("PUT /v1/engine/{engine}/connections/{name}/grants", controller.SetGrant)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/grants", controller.DeleteGrant)

	// SQL-only: schema, row preview and the console ops.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/schema", controller.GetSchema)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/tables/{table}/rows", controller.GetRows)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/query", controller.PostQuery)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/alter", controller.PostAlter)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/rollback", controller.PostRollback)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/analyze", controller.PostAnalyze)

	// Changes: SQL and MongoDB keep separate queues behind one route.
	mux.HandleFunc("GET /v1/engine/{engine}/changes", controller.ListChanges)
	mux.HandleFunc("POST /v1/engine/{engine}/changes", controller.SubmitChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/batch", controller.SubmitChangeBatch)
	mux.HandleFunc("GET /v1/engine/{engine}/changes/{id}", controller.GetChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/approve", controller.ApproveChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/apply", controller.ApplyChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/reject", controller.RejectChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/schedule", controller.ScheduleChange)

	// Memory-cache-only: keys, values, removal and emptying the database.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/keys", controller.ListKeys)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/keys", controller.DeleteKeys)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/value", controller.GetValue)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/flush", controller.FlushDB)

	// MongoDB-only: databases, collections and read requests.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/databases", controller.ListDatabases)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/collections", controller.ListCollections)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/requests", controller.PostRequest)

	mux.Handle("GET /assets/", http.HandlerFunc(controller.GetAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", controller.GetConsole)                     // the operator console (SPA shell)

	// Scheduling: an external cron (or the in-process scheduler) drives due
	// changes through the same handler.
	mux.HandleFunc("POST /internal/tick", controller.PostTick)

	// Instances address a connection by its stable id, across families. The
	// family routes live under /v1/engine/{engine}/..., so these paths never
	// collide with the {engine} wildcard.
	mux.HandleFunc("GET /v1/instances", controller.ListInstances)
	mux.HandleFunc("GET /v1/instances/{id}", controller.GetInstance)
	mux.HandleFunc("GET /v1/instances/{id}/changes", controller.ListInstanceChanges)
	mux.HandleFunc("POST /v1/instances/{id}/password", controller.RotatePassword)

	return withMiddleware(withAuth(verifier, controller.WithDeps(mux, deps)), deps.Log)
}
