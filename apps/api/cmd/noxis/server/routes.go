package server

import (
	"log/slog"
	"net/http"
	"strings"

	"noxis/cmd/noxis/server/middlewares"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
)

// route is one HTTP endpoint: its method and path, the handler, whether it
// bypasses the bearer token, and any middleware that only it needs.
type route struct {
	Method      string
	Endpoint    string
	Handler     http.Handler
	Public      bool
	Middlewares []func(http.Handler) http.Handler
}

// routes lists the whole HTTP surface in one place, so a reviewer can see the
// method, path and access rule of every endpoint without reading a wall of
// registrations.
func routes(d app, cfg config.Config) []route {
	const api = constants.APIPrefix
	return []route{
		{Method: http.MethodGet, Endpoint: "/healthz", Handler: http.HandlerFunc(getHealthz)},
		{Method: http.MethodGet, Endpoint: api + "/config/drivers", Handler: http.HandlerFunc(d.Drivers.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/config/levels", Handler: http.HandlerFunc(d.Levels.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/auth/config", Handler: http.HandlerFunc(d.AuthConfig.Serve), Public: true},
		{Method: http.MethodPost, Endpoint: api + "/auth/login", Handler: http.HandlerFunc(d.Login.Serve), Public: true},
		{Method: http.MethodGet, Endpoint: api + "/me", Handler: http.HandlerFunc(d.Me.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/auth/logout", Handler: http.HandlerFunc(d.Logout.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/auth/password", Handler: http.HandlerFunc(d.ChangePass.Serve)},

		// Connections: one generic route per operation, gated by the {engine} path parameter through the module registry.
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections", Handler: http.HandlerFunc(d.ConnectionList.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections", Handler: http.HandlerFunc(d.SaveConnection.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}", Handler: http.HandlerFunc(d.Connection.Serve)},
		{Method: http.MethodDelete, Endpoint: api + "/engine/{engine}/connections/{name}", Handler: http.HandlerFunc(d.DeleteConnection.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/test", Handler: http.HandlerFunc(d.TestConnection.Serve)},

		// Permissions: per-connection RBAC policy (users and groups), managed by admins.
		{Method: http.MethodGet, Endpoint: api + "/config/members", Handler: http.HandlerFunc(d.Members.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/access", Handler: http.HandlerFunc(d.AccessMatrix.Serve)},

		// Access requests: a user asks, an admin or the instance's admin decides.
		{Method: http.MethodPost, Endpoint: api + "/access-requests", Handler: http.HandlerFunc(d.AccessCreate.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/access-requests", Handler: http.HandlerFunc(d.AccessList.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/access-requests/{id}/approve", Handler: http.HandlerFunc(d.AccessApprove.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/access-requests/{id}/reject", Handler: http.HandlerFunc(d.AccessReject.Serve)},

		// Users: the admin directory.
		{Method: http.MethodGet, Endpoint: api + "/users", Handler: http.HandlerFunc(d.UserList.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/users/{subject}", Handler: http.HandlerFunc(d.UserGet.Serve)},
		{Method: http.MethodPatch, Endpoint: api + "/users/{subject}", Handler: http.HandlerFunc(d.UserUpdate.Serve)},
		// The instance-scoped grant routes live under the engine path, so a family mismatch is reported like any other connection error.
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/permissions", Handler: http.HandlerFunc(d.ListPermissions.Serve)},
		{Method: http.MethodPut, Endpoint: api + "/engine/{engine}/connections/{name}/permissions", Handler: http.HandlerFunc(d.SetPermission.Serve)},
		{Method: http.MethodDelete, Endpoint: api + "/engine/{engine}/connections/{name}/permissions", Handler: http.HandlerFunc(d.DeletePermission.Serve)},

		// SQL-only: schema, row preview and the console ops.
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/schema", Handler: http.HandlerFunc(d.Schema.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/erd", Handler: http.HandlerFunc(d.ERD.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/tables/{table}/rows", Handler: http.HandlerFunc(d.Rows.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/query", Handler: http.HandlerFunc(d.Op.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/alter", Handler: http.HandlerFunc(d.Op.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/rollback", Handler: http.HandlerFunc(d.Op.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/analyze", Handler: http.HandlerFunc(d.Op.Serve)},

		// Changes: SQL and MongoDB keep separate queues behind one route.
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/changes", Handler: http.HandlerFunc(d.ChangeList.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/changes", Handler: http.HandlerFunc(d.ChangeSubmit.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/changes/{id}", Handler: http.HandlerFunc(d.ChangeGet.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/changes/{id}/approve", Handler: http.HandlerFunc(d.ChangeApprove.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/changes/{id}/apply", Handler: http.HandlerFunc(d.ChangeApply.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/changes/{id}/reject", Handler: http.HandlerFunc(d.ChangeReject.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/changes/{id}/schedule", Handler: http.HandlerFunc(d.ChangeSchedule.Serve)},

		// Memory-cache-only: keys, values, removal and emptying the database.
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/keys", Handler: http.HandlerFunc(d.CacheKeys.Serve)},
		{Method: http.MethodDelete, Endpoint: api + "/engine/{engine}/connections/{name}/keys", Handler: http.HandlerFunc(d.CacheDel.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/value", Handler: http.HandlerFunc(d.CacheVal.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/flush", Handler: http.HandlerFunc(d.CacheFlsh.Serve)},

		// MongoDB-only: databases, collections and read requests.
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/databases", Handler: http.HandlerFunc(d.MongoDbs.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{engine}/connections/{name}/collections", Handler: http.HandlerFunc(d.MongoColls.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{engine}/connections/{name}/requests", Handler: http.HandlerFunc(d.MongoRun.Serve)},

		// Instances address a connection by its stable id, across families. The family routes live under {api}/engine/{engine}/..., so these paths never collide with the {engine} wildcard.
		{Method: http.MethodGet, Endpoint: api + "/instances", Handler: http.HandlerFunc(d.Instances.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/instances/{id}", Handler: http.HandlerFunc(d.Instance.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/instances/{id}/password", Handler: http.HandlerFunc(d.RotatePassword.Serve)},

		// Console and probes.
		{Method: http.MethodGet, Endpoint: "/assets/", Handler: http.HandlerFunc(getAssets)}, // embedded FS keeps the assets/ prefix
		{Method: http.MethodGet, Endpoint: "/", Handler: http.HandlerFunc(getConsole)},       // the operator console (SPA shell)

		// Scheduling uses an authenticated external callback; no in-process ticker runs.
		{
			Method:   http.MethodPost,
			Endpoint: "/internal/callback/scheduler",
			Handler:  http.HandlerFunc(d.Tick.Serve),
			Middlewares: []func(http.Handler) http.Handler{
				func(next http.Handler) http.Handler {
					return middlewares.WithSchedulerAuth(cfg.SchedulerKey, cfg.SchedulerSecret, next)
				},
			},
		},
	}
}

// newHandler wires the route table to a mux, applying auth to every API route
// that is not explicitly public and the request log to every route.
func newHandler(d app, cfg config.Config, verifier interfaces.Verifier, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes(d, cfg) {
		handler := middlewares.RequestLog(rt.Handler, log)
		if !rt.Public && strings.HasPrefix(rt.Endpoint, constants.APIPrefix) {
			handler = middlewares.Auth(verifier, log)(handler)
		}
		for i := len(rt.Middlewares) - 1; i >= 0; i-- {
			handler = rt.Middlewares[i](handler)
		}
		mux.Handle(rt.Method+" "+rt.Endpoint, handler)
	}
	return mux
}
