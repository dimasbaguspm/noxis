package master

import (
	"log/slog"
	"net/http"

	"noxis/internal/repository"
	rpc "noxis/internal/repository/grpc"
)

// newHandler wires the REST routes, middleware and the control core.
func newHandler(ctrl *rpc.Server, repo *repository.SQLite, log *slog.Logger) http.Handler {
	h := &controller{ctrl: ctrl, repo: repo, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", h.listStatus)
	mux.HandleFunc("GET /v1/nodes/{node}", h.getNode)
	mux.HandleFunc("PUT /v1/nodes/{node}", h.putNode)
	mux.HandleFunc("POST /v1/nodes/{node}/query", h.query)
	mux.HandleFunc("POST /v1/nodes/{node}/register", h.register)
	return withMiddleware(mux, log)
}
