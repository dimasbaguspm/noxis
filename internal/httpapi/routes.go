// Package httpapi is the REST transport: the route table, the middlewares and the handlers. It
// names no adapter and no wiring — the composition root hands it a route table's worth of use cases
// and it answers HTTP.
package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"noxis/internal/constants"
	"noxis/internal/httpapi/handlers"
	"noxis/internal/httpapi/middlewares"
)

// Deps is what the transport is built from: the logger, the authenticator and one handler set.
type Deps struct {
	Log      *slog.Logger
	Auth     *middlewares.Authenticator
	Handlers *handlers.Handlers
}

// Route is one line of the API. A route that is not Public names the scopes it needs, so the whole
// access policy fits on one screen and no handler checks a scope itself.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
	Scopes  []string
	Public  bool
}

// NewMux registers the route table and wraps it once.
func NewMux(d Deps) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range Routes(d) {
		handler := rt.Handler
		if !rt.Public {
			handler = d.Auth.Require(rt.Scopes, handler)
		}
		listed := fmt.Sprintf("%s %s", rt.Method, rt.Path)
		mux.Handle(listed, middlewares.Span(listed)(handler))
	}
	return middlewares.Chain(mux,
		middlewares.Recover(d.Log),
		middlewares.CORS(constants.ConsoleOrigins),
		middlewares.Access(d.Log),
	)
}

// Routes is the whole API in one slice. It is exported because the contract test walks it: a route
// that is not in the OpenAPI document is a route nobody can call on purpose.
//
// The paths are written here, as literals, because this is where the route is: a path that lives in
// `constants` has a second place to be wrong and no second reader to justify it.
func Routes(d Deps) []Route {
	read := []string{constants.ScopeRead}
	admin := []string{constants.ScopeAdmin}
	propose := []string{constants.ScopePropose}
	target := "/v1/targets/{ref}"
	proposal := "/v1/proposals/{ref}"
	return []Route{
		{Method: http.MethodGet, Path: "/healthz", Handler: d.Handlers.Healthz, Public: true},
		{Method: http.MethodGet, Path: "/openapi.json", Handler: OpenAPIJSON, Public: true},
		{Method: http.MethodGet, Path: "/openapi.yaml", Handler: OpenAPIYAML, Public: true},
		{Method: http.MethodGet, Path: "/v1/whoami", Handler: d.Handlers.Whoami},

		{Method: http.MethodGet, Path: "/v1/targets", Handler: d.Handlers.ListTargets, Scopes: read},
		{Method: http.MethodPost, Path: "/v1/targets", Handler: d.Handlers.CreateTarget, Scopes: admin},
		{Method: http.MethodGet, Path: target, Handler: d.Handlers.GetTarget, Scopes: read},
		{Method: http.MethodPatch, Path: target, Handler: d.Handlers.UpdateTarget, Scopes: admin},
		{Method: http.MethodDelete, Path: target, Handler: d.Handlers.ArchiveTarget, Scopes: admin},
		{Method: http.MethodPost, Path: target + "/probe", Handler: d.Handlers.ProbeTarget, Scopes: read},
		{Method: http.MethodGet, Path: target + "/capabilities", Handler: d.Handlers.TargetCapabilities, Scopes: read},

		{Method: http.MethodPost, Path: target + "/query", Handler: d.Handlers.RunQuery, Scopes: read},
		{Method: http.MethodGet, Path: target + "/databases", Handler: d.Handlers.ListDatabases, Scopes: read},
		{Method: http.MethodGet, Path: target + "/databases/{db}/tables", Handler: d.Handlers.ListTables, Scopes: read},
		{Method: http.MethodGet, Path: target + "/databases/{db}/tables/{table}", Handler: d.Handlers.ShowTable, Scopes: read},

		{Method: http.MethodPost, Path: target + "/proposals", Handler: d.Handlers.CreateProposal, Scopes: propose},
		{Method: http.MethodGet, Path: "/v1/proposals", Handler: d.Handlers.ListProposals, Scopes: read},
		{Method: http.MethodGet, Path: proposal, Handler: d.Handlers.GetProposal, Scopes: read},
		{Method: http.MethodPost, Path: proposal + "/rehearse", Handler: d.Handlers.RehearseProposal, Scopes: propose},
		{Method: http.MethodPost, Path: proposal + "/approve", Handler: d.Handlers.ApproveProposal, Scopes: []string{constants.ScopeApprove}},
		{Method: http.MethodPost, Path: proposal + "/reject", Handler: d.Handlers.RejectProposal, Scopes: []string{constants.ScopeApprove}},
		{Method: http.MethodPost, Path: proposal + "/apply", Handler: d.Handlers.ApplyProposal, Scopes: []string{constants.ScopeApply}},
		{Method: http.MethodGet, Path: proposal + "/events", Handler: d.Handlers.ProposalEvents, Scopes: read},
	}
}
