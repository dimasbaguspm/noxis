package main

import (
	"net/http"

	"noxis/cmd/http/middlewares"
	"noxis/internal/constants"
)

// route is one line of the API. A route that is not Public names the scopes it needs, so
// the whole access policy fits on one screen and no handler checks a scope itself.
type route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
	Scopes  []string
	Public  bool
}

// newMux registers the route table and wraps it once.
func newMux(d deps) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes(d) {
		handler := rt.Handler
		if !rt.Public {
			handler = d.auth.Require(rt.Scopes, handler)
		}
		mux.HandleFunc(rt.Method+" "+rt.Path, handler)
	}
	return middlewares.Chain(mux,
		middlewares.Recover(d.log),
		middlewares.CORS(d.cfg.CORSOrigins),
		middlewares.Access(d.log),
	)
}

func routes(d deps) []route {
	return []route{
		{Method: http.MethodGet, Path: constants.PathHealthz, Handler: d.handlers.Healthz, Public: true},
		{Method: http.MethodGet, Path: constants.PathWhoami, Handler: d.handlers.Whoami},
	}
}
