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
	"noxis/internal/services/sql"
)

// newHandler wires routes to handlers; dependencies ride the request ctx.
func newHandler(store *sqlite.Store, log *slog.Logger) http.Handler {
	deps := controller.Deps{
		Connections: store,
		Changes:     store,
		SQL:         sql.New(),
		Log:         log,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", controller.GetHealthz)
	mux.HandleFunc("GET /v1/connections/status/stream", controller.StreamConnectionStatus)
	mux.HandleFunc("GET /v1/connections", controller.ListConnections)
	mux.HandleFunc("POST /v1/connections", controller.SaveConnection)
	mux.HandleFunc("GET /v1/connections/{name}", controller.GetConnection)
	mux.HandleFunc("DELETE /v1/connections/{name}", controller.DeleteConnection)
	mux.HandleFunc("POST /v1/connections/{name}/test", controller.TestConnection)
	mux.HandleFunc("GET /v1/connections/{name}/schema", controller.GetSchema)
	mux.HandleFunc("GET /v1/connections/{name}/tables/{table}/rows", controller.GetRows)
	mux.HandleFunc("GET /v1/connections/{name}/stats", controller.GetStats)
	mux.HandleFunc("GET /v1/connections/{name}/stats/stream", controller.StreamStats)
	mux.HandleFunc("GET /v1/changes", controller.ListChanges)
	mux.HandleFunc("POST /v1/changes", controller.SubmitChange)
	mux.HandleFunc("GET /v1/changes/{id}", controller.GetChange)
	mux.HandleFunc("POST /v1/changes/{id}/reject", controller.RejectChange)
	mux.HandleFunc("POST /v1/changes/{id}/approve", controller.ApproveChange)
	mux.HandleFunc("POST /v1/changes/{id}/apply", controller.ApplyChange)
	mux.HandleFunc("POST /v1/connections/{name}/query", controller.PostQuery)
	mux.HandleFunc("POST /v1/connections/{name}/alter", controller.PostAlter)
	mux.HandleFunc("POST /v1/connections/{name}/rollback", controller.PostRollback)
	mux.HandleFunc("POST /v1/connections/{name}/analytics", controller.PostAnalytics)
	mux.Handle("GET /assets/", http.HandlerFunc(controller.GetAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", controller.GetConsole)                     // the operator console (SPA shell)

	return withMiddleware(controller.WithDeps(mux, deps), log)
}
