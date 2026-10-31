package server

import (
	"log/slog"
	"net/http"
	"strings"

	"noxis/cmd/noxis/server/middlewares"
	"noxis/internal/common"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/usecases/access"
	"noxis/internal/usecases/apply"
	"noxis/internal/usecases/auth"
	"noxis/internal/usecases/cache"
	"noxis/internal/usecases/change"
	"noxis/internal/usecases/connection"
	"noxis/internal/usecases/driver"
	"noxis/internal/usecases/instance"
	"noxis/internal/usecases/metadata"
	"noxis/internal/usecases/mongo"
	"noxis/internal/usecases/permission"
	"noxis/internal/usecases/schema"
	"noxis/internal/usecases/search"
	"noxis/internal/usecases/user"
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
func routes(w Wiring, cfg config.Config) []route {
	const api = constants.APIPrefix
	return []route{
		{Method: http.MethodGet, Endpoint: "/healthz", Handler: http.HandlerFunc(getHealthz)},
		{Method: http.MethodGet, Endpoint: api + "/config/drivers", Handler: http.HandlerFunc(driver.Drivers{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/config/levels", Handler: http.HandlerFunc(metadata.Levels{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/auth/config", Handler: http.HandlerFunc(auth.Config{RegistryDeps: w.Usecases, Auth: w.Auth}.Serve), Public: true},
		{Method: http.MethodPost, Endpoint: api + "/auth/login", Handler: http.HandlerFunc(auth.Login{RegistryDeps: w.Usecases}.Serve), Public: true},
		{Method: http.MethodGet, Endpoint: api + "/me", Handler: http.HandlerFunc(auth.Me{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/auth/logout", Handler: http.HandlerFunc(auth.Logout{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/auth/password", Handler: http.HandlerFunc(auth.Password{RegistryDeps: w.Usecases}.Serve)},

		// Connections: one generic route per operation, gated by the {driver} path parameter through the module registry.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections", Handler: http.HandlerFunc(connection.List{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections", Handler: http.HandlerFunc(connection.Save{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}", Handler: http.HandlerFunc(connection.Get{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodDelete, Endpoint: api + "/engine/{driver}/connections/{name}", Handler: http.HandlerFunc(connection.Delete{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/{name}/test", Handler: http.HandlerFunc(connection.Test{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/probe", Handler: http.HandlerFunc(connection.Probe{RegistryDeps: w.Usecases}.Serve)},

		// Permissions: per-connection RBAC policy (users and groups), managed by admins.
		{Method: http.MethodGet, Endpoint: api + "/config/members", Handler: http.HandlerFunc(permission.Members{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/access", Handler: http.HandlerFunc(permission.Matrix{RegistryDeps: w.Usecases}.Serve)},

		// Access requests: a user asks, an admin or the instance's admin decides.
		{Method: http.MethodPost, Endpoint: api + "/access-requests", Handler: http.HandlerFunc(access.Create{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/access-requests", Handler: http.HandlerFunc(access.List{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/access-requests/{id}/approve", Handler: http.HandlerFunc(access.Approve{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/access-requests/{id}/reject", Handler: http.HandlerFunc(access.Reject{RegistryDeps: w.Usecases}.Serve)},

		// Users: the admin directory.
		{Method: http.MethodGet, Endpoint: api + "/users", Handler: http.HandlerFunc(user.List{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/users/{subject}", Handler: http.HandlerFunc(user.Get{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPatch, Endpoint: api + "/users/{subject}", Handler: http.HandlerFunc(user.Update{RegistryDeps: w.Usecases}.Serve)},
		// The instance-scoped grant routes live under the engine path, so a driver mismatch is reported like any other connection error.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/permissions", Handler: http.HandlerFunc(permission.List{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPut, Endpoint: api + "/engine/{driver}/connections/{name}/permissions", Handler: http.HandlerFunc(permission.Set{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodDelete, Endpoint: api + "/engine/{driver}/connections/{name}/permissions", Handler: http.HandlerFunc(permission.Delete{RegistryDeps: w.Usecases}.Serve)},

		// SQL-only: schema, row preview and the console ops.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/schema", Handler: http.HandlerFunc(schema.Tables{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/erd", Handler: http.HandlerFunc(schema.ERD{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/info", Handler: http.HandlerFunc(schema.Info{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/tables/{table}/rows", Handler: http.HandlerFunc(schema.Rows{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/{name}/operations", Handler: http.HandlerFunc(schema.Op{RegistryDeps: w.Usecases}.Serve)},

		// Changes: SQL and MongoDB keep separate queues behind one route.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/changes", Handler: http.HandlerFunc(change.List{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/changes", Handler: http.HandlerFunc(change.Submit{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/changes/{id}", Handler: http.HandlerFunc(change.Get{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/changes/{id}/approve", Handler: http.HandlerFunc(change.Approve{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/changes/{id}/apply", Handler: http.HandlerFunc(apply.Apply{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/changes/{id}/reject", Handler: http.HandlerFunc(change.Reject{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/changes/{id}/schedule", Handler: http.HandlerFunc(change.Schedule{RegistryDeps: w.Usecases}.Serve)},

		// Memory-cache-only: keys, values, removal and emptying the database.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/keys", Handler: http.HandlerFunc(cache.Keys{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodDelete, Endpoint: api + "/engine/{driver}/connections/{name}/keys", Handler: http.HandlerFunc(cache.Delete{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/value", Handler: http.HandlerFunc(cache.Value{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/{name}/flush", Handler: http.HandlerFunc(cache.Flush{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/stats", Handler: http.HandlerFunc(cache.Info{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/{name}/keys/ttl", Handler: http.HandlerFunc(cache.TTL{RegistryDeps: w.Usecases}.Serve)},

		// MongoDB-only: databases, collections and read requests.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/databases", Handler: http.HandlerFunc(mongo.Databases{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/collections", Handler: http.HandlerFunc(mongo.Collections{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/{name}/requests", Handler: http.HandlerFunc(mongo.Run{RegistryDeps: w.Usecases}.Serve)},
		// Search platforms: what they index, and one query against a target.
		{Method: http.MethodGet, Endpoint: api + "/engine/{driver}/connections/{name}/targets", Handler: http.HandlerFunc(search.Targets{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/engine/{driver}/connections/{name}/search", Handler: http.HandlerFunc(search.Run{RegistryDeps: w.Usecases}.Serve)},

		// Instances address a connection by its stable id, across drivers. The engine routes live under {api}/engine/{driver}/..., so these paths never collide with the {driver} wildcard.
		{Method: http.MethodGet, Endpoint: api + "/instances", Handler: http.HandlerFunc(instance.List{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/instances/counts", Handler: http.HandlerFunc(instance.Counts{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodGet, Endpoint: api + "/instances/{id}", Handler: http.HandlerFunc(instance.Get{RegistryDeps: w.Usecases}.Serve)},
		{Method: http.MethodPost, Endpoint: api + "/instances/{id}/password", Handler: http.HandlerFunc(instance.RotatePassword{RegistryDeps: w.Usecases}.Serve)},

		// Console and probes.
		{Method: http.MethodGet, Endpoint: "/assets/", Handler: http.HandlerFunc(getAssets)}, // embedded FS keeps the assets/ prefix
		{Method: http.MethodGet, Endpoint: "/", Handler: http.HandlerFunc(getConsole)},       // the operator console (SPA shell)

		// Scheduling uses an authenticated external callback; no in-process ticker runs.
		{
			Method:   http.MethodPost,
			Endpoint: "/internal/callback/scheduler",
			Handler:  http.HandlerFunc(apply.Tick{RegistryDeps: w.Usecases}.Serve),
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
// withDriver attaches the {driver} path value to the request context, so a use case resolves its module without reading the route itself.
func withDriver(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if driver := r.PathValue("driver"); driver != "" {
			r = r.WithContext(common.WithModuleDriver(r.Context(), driver))
		}
		h.ServeHTTP(w, r)
	})
}

func newHandler(w Wiring, cfg config.Config, verifier interfaces.Verifier, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes(w, cfg) {
		handler := middlewares.RequestLog(withDriver(rt.Handler), log)
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
