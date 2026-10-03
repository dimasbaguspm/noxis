// noxis control plane API
//
// @title       noxis control plane API
// @version     0.1.0
// @description Master REST API: frontends read node state here; the master
// publishes desired-state events to slave workers over Kafka (not part of
// this API).
package master

import (
	"log/slog"
	"net/http"

	"noxis/cmd/noxis/master/controller"
	control "noxis/internal"
)

// newHandler wires every route to its controller handler, then middleware.
func newHandler(ctrl *control.Control, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", controller.GetHealthz)
	mux.HandleFunc("GET /v1/status", controller.GetStatus)
	mux.HandleFunc("GET /v1/nodes/{node}/spec", controller.GetNodeSpec)
	mux.HandleFunc("PUT /v1/nodes/{node}/spec", controller.PutNodeSpec)
	mux.HandleFunc("POST /v1/nodes/{node}/doorman/query", controller.PostQuery)
	mux.HandleFunc("POST /v1/nodes/{node}/doorman/alter", controller.PostAlter)
	mux.HandleFunc("POST /v1/nodes/{node}/doorman/rollback", controller.PostRollback)
	mux.HandleFunc("POST /v1/nodes/{node}/doorman/analytics", controller.PostAnalytics)
	mux.HandleFunc("POST /v1/nodes/{node}/register", controller.PostRegister)

	return withMiddleware(controller.WithDeps(mux, controller.Deps{Ctrl: ctrl, Log: log}), log)
}
