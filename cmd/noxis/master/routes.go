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
	modules.Register(mux, modules.Deps{Ctrl: ctrl, Repo: repo, Log: log})
	return withMiddleware(mux, log)
}
