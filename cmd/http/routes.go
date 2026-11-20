package main

import (
	"net/http"

	"noxis/cmd/http/middlewares"
	"noxis/internal/constants"
)

// route is one line of the API. A route that is not Public names the scopes it needs, so the
// whole access policy fits on one screen and no handler checks a scope itself.
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
		middlewares.CORS(constants.ConsoleOrigins),
		middlewares.Access(d.log),
	)
}

func routes(d deps) []route {
	read := []string{constants.ScopeRead}
	admin := []string{constants.ScopeAdmin}
	one := constants.PathTargets + "/{ref}"
	propose := []string{constants.ScopePropose}
	proposalOne := constants.PathProposals + "/{ref}"
	return []route{
		{Method: http.MethodGet, Path: constants.PathHealthz, Handler: d.handlers.Healthz, Public: true},
		{Method: http.MethodGet, Path: constants.PathWhoami, Handler: d.handlers.Whoami},

		{Method: http.MethodGet, Path: constants.PathTargets, Handler: d.handlers.ListTargets, Scopes: read},
		{Method: http.MethodPost, Path: constants.PathTargets, Handler: d.handlers.CreateTarget, Scopes: admin},
		{Method: http.MethodGet, Path: one, Handler: d.handlers.GetTarget, Scopes: read},
		{Method: http.MethodPatch, Path: one, Handler: d.handlers.UpdateTarget, Scopes: admin},
		{Method: http.MethodDelete, Path: one, Handler: d.handlers.ArchiveTarget, Scopes: admin},
		{Method: http.MethodPost, Path: one + "/probe", Handler: d.handlers.ProbeTarget, Scopes: read},
		{Method: http.MethodGet, Path: one + "/capabilities", Handler: d.handlers.TargetCapabilities, Scopes: read},

		{Method: http.MethodPost, Path: constants.PathTargets + "/{ref}/query", Handler: d.handlers.RunQuery, Scopes: read},
		{Method: http.MethodGet, Path: constants.PathTargets + "/{ref}/databases", Handler: d.handlers.ListDatabases, Scopes: read},
		{Method: http.MethodGet, Path: constants.PathTargets + "/{ref}/databases/{db}/tables", Handler: d.handlers.ListTables, Scopes: read},
		{Method: http.MethodGet, Path: constants.PathTargets + "/{ref}/databases/{db}/tables/{table}", Handler: d.handlers.ShowTable, Scopes: read},

		{Method: http.MethodPost, Path: constants.PathTargets + "/{ref}/proposals", Handler: d.handlers.CreateProposal, Scopes: propose},
		{Method: http.MethodGet, Path: constants.PathProposals, Handler: d.handlers.ListProposals, Scopes: read},
		{Method: http.MethodGet, Path: proposalOne, Handler: d.handlers.GetProposal, Scopes: read},
		{Method: http.MethodPost, Path: proposalOne + "/rehearse", Handler: d.handlers.RehearseProposal, Scopes: propose},
		{Method: http.MethodPost, Path: proposalOne + "/approve", Handler: d.handlers.ApproveProposal, Scopes: []string{constants.ScopeApprove}},
		{Method: http.MethodPost, Path: proposalOne + "/reject", Handler: d.handlers.RejectProposal, Scopes: []string{constants.ScopeApprove}},
		{Method: http.MethodPost, Path: proposalOne + "/apply", Handler: d.handlers.ApplyProposal, Scopes: []string{constants.ScopeApply}},
		{Method: http.MethodGet, Path: proposalOne + "/events", Handler: d.handlers.ProposalEvents, Scopes: read},
	}
}
