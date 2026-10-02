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

	"noxis/cmd/noxis/master/modules"
	"noxis/internal/repository"
	rpc "noxis/internal/services/grpc"
)

// newHandler assembles the REST API from feature modules + middleware.
func newHandler(ctrl *rpc.Server, repo *repository.SQLite, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	d := modules.Deps{Ctrl: ctrl, Repo: repo, Log: log}
	modules.Register(mux, d)
	return withMiddleware(modules.WithDeps(mux, d), log)
}
