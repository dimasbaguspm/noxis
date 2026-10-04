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
	"noxis/internal/repositories/sqlite"
	"noxis/internal/services/memcache"
	"noxis/internal/services/sql"
)

// newHandler wires routes to handlers; dependencies ride the request ctx.
func newHandler(store *sqlite.Store, log *slog.Logger) http.Handler {
	deps := controller.Deps{
		Connections: store,
		Changes:     store,
		SQL:         sql.New(),
		Memcache:    memcache.New(),
		Log:         log,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", controller.GetHealthz)
	mux.HandleFunc("GET /v1/connections/status/stream", controller.StreamConnectionStatus)
	mux.HandleFunc("GET /v1/sql/connections", controller.ListConnections)
	mux.HandleFunc("POST /v1/sql/connections", controller.SaveConnection)
	mux.HandleFunc("GET /v1/sql/connections/{name}", controller.GetConnection)
	mux.HandleFunc("DELETE /v1/sql/connections/{name}", controller.DeleteConnection)
	mux.HandleFunc("POST /v1/sql/connections/{name}/test", controller.TestConnection)
	mux.HandleFunc("GET /v1/sql/connections/{name}/schema", controller.GetSchema)
	mux.HandleFunc("GET /v1/sql/connections/{name}/tables/{table}/rows", controller.GetRows)
	mux.HandleFunc("GET /v1/sql/connections/{name}/stats", controller.GetStats)
	mux.HandleFunc("GET /v1/sql/connections/{name}/stats/stream", controller.StreamStats)
	mux.HandleFunc("GET /v1/sql/changes", controller.ListChanges)
	mux.HandleFunc("POST /v1/sql/changes", controller.SubmitChange)
	mux.HandleFunc("GET /v1/sql/changes/{id}", controller.GetChange)
	mux.HandleFunc("POST /v1/sql/changes/{id}/reject", controller.RejectChange)
	mux.HandleFunc("POST /v1/sql/changes/{id}/approve", controller.ApproveChange)
	mux.HandleFunc("POST /v1/sql/changes/{id}/apply", controller.ApplyChange)
	mux.HandleFunc("POST /v1/sql/connections/{name}/query", controller.PostQuery)
	mux.HandleFunc("POST /v1/sql/connections/{name}/alter", controller.PostAlter)
	mux.HandleFunc("POST /v1/sql/connections/{name}/rollback", controller.PostRollback)
	mux.HandleFunc("POST /v1/sql/connections/{name}/analytics", controller.PostAnalytics)

	// The memory-cache namespace (Memcache and Valkey): the same connection
	// lifecycle, plus keys, values, removal and INFO stats.
	mux.HandleFunc("GET /v1/memcache/connections", controller.ListMemcacheConnections)
	mux.HandleFunc("POST /v1/memcache/connections", controller.SaveMemcacheConnection)
	mux.HandleFunc("GET /v1/memcache/connections/{name}", controller.GetMemcacheConnection)
	mux.HandleFunc("DELETE /v1/memcache/connections/{name}", controller.DeleteMemcacheConnection)
	mux.HandleFunc("POST /v1/memcache/connections/{name}/test", controller.TestMemcacheConnection)
	mux.HandleFunc("GET /v1/memcache/connections/{name}/keys", controller.ListMemcacheKeys)
	mux.HandleFunc("DELETE /v1/memcache/connections/{name}/keys", controller.DeleteMemcacheKeys)
	mux.HandleFunc("GET /v1/memcache/connections/{name}/value", controller.GetMemcacheValue)
	mux.HandleFunc("POST /v1/memcache/connections/{name}/flush", controller.FlushMemcacheDB)
	mux.HandleFunc("GET /v1/memcache/connections/{name}/stats", controller.GetMemcacheStats)
	mux.HandleFunc("GET /v1/memcache/connections/{name}/stats/stream", controller.StreamMemcacheStats)
	mux.Handle("GET /assets/", http.HandlerFunc(controller.GetAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", controller.GetConsole)                     // the operator console (SPA shell)

	return withMiddleware(controller.WithDeps(mux, deps), log)
}
