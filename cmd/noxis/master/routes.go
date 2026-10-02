// noxis control plane API
//
// @title       noxis control plane API
// @version     0.1.0
// @description Master REST API: frontends read node state here; the master
// publishes desired-state events to slave workers over an internal gRPC
// transport (not part of this API).
package master

import (
	"log/slog"
	"net/http"

	"noxis/cmd/noxis/master/controller"
	"noxis/internal/repository"
	rpc "noxis/internal/services/grpc"
)

// newHandler wires every route to its controller handler, then middleware.
func newHandler(ctrl *rpc.Server, repo *repository.SQLite, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	d := controller.Deps{Ctrl: ctrl, Repo: repo, Log: log}

	mux.HandleFunc("GET /healthz", controller.GetHealthz)
	mux.HandleFunc("GET /readyz", controller.GetReadyz)
	mux.HandleFunc("GET /v1/status", controller.GetStatus)
	mux.HandleFunc("GET /v1/nodes/{node}/spec", controller.GetNodeSpec)
	mux.HandleFunc("PUT /v1/nodes/{node}/spec", controller.PutNodeSpec)
	mux.HandleFunc("POST /v1/nodes/{node}/query", controller.PostQuery)
	mux.HandleFunc("POST /v1/nodes/{node}/register", controller.PostRegister)

	return withMiddleware(controller.WithDeps(mux, d), log)
}
