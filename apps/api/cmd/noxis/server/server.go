// Package server is the noxis REST API and operator console: the transport
// wiring. Swagger annotations live on the use case Serve methods.
//
// @title       noxis API
// @version     0.1.0
// @description SQL console API: manage database connections and run
// verified reads, writes, dry-run rollbacks and analytics against them.
package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"noxis/cmd/noxis/server/middlewares"
	"noxis/internal/common"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/auth"
	"noxis/internal/repositories"
	"noxis/internal/repositories/sqlite"
)

const shutdownTimeout = 10 * time.Second

// newDeps assembles the runtime container once: the stores, the registry (with the auth service) and the transport values, then builds every use case.
func newDeps(store *sqlite.Store, cfg config.Config, authSvc *auth.Service) app {
	return newApp(sources{
		Repositories: repositories.Default(store),
		Modules:      nil, // built in newApp, where the instruments exist
		Auth:         cfg.AuthInfo(),
		AuthService:  authSvc,
	})
}

// newHandler wires routes straight to use case methods.
func newHandler(d app, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	// The public API is versioned and mounted under /external; internal triggers live under /internal; the console shell and health probes stay at the root.
	const api = constants.APIPrefix

	mux.HandleFunc("GET /healthz", getHealthz)
	mux.HandleFunc("GET "+api+"/config/drivers", d.Drivers.Serve)
	mux.HandleFunc("GET "+api+"/config/levels", d.Levels.Serve)
	mux.HandleFunc("GET "+api+"/auth/config", d.AuthConfig.Serve)
	mux.HandleFunc("POST "+api+"/auth/login", d.Login.Serve)
	mux.HandleFunc("GET "+api+"/me", d.Me.Serve)
	mux.HandleFunc("POST "+api+"/auth/logout", d.Logout.Serve)
	mux.HandleFunc("POST "+api+"/auth/password", d.ChangePass.Serve)

	// Connections: one generic route per operation, gated by the {engine} path parameter through the module registry.
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections", d.ConnectionList.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections", d.SaveConnection.Serve)
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}", d.Connection.Serve)
	mux.HandleFunc("DELETE "+api+"/engine/{engine}/connections/{name}", d.DeleteConnection.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/test", d.TestConnection.Serve)

	// Permissions: per-connection RBAC policy (users and groups), managed by admins.
	mux.HandleFunc("GET "+api+"/config/members", d.Members.Serve)
	mux.HandleFunc("GET "+api+"/access", d.AccessMatrix.Serve)

	// Access requests: a user asks, an admin or the instance's admin decides.
	mux.HandleFunc("POST "+api+"/access-requests", d.AccessCreate.Serve)
	mux.HandleFunc("GET "+api+"/access-requests", d.AccessList.Serve)
	mux.HandleFunc("POST "+api+"/access-requests/{id}/approve", d.AccessApprove.Serve)
	mux.HandleFunc("POST "+api+"/access-requests/{id}/reject", d.AccessReject.Serve)

	// Users: the admin directory.
	mux.HandleFunc("GET "+api+"/users", d.UserList.Serve)
	mux.HandleFunc("GET "+api+"/users/{subject}", d.UserGet.Serve)
	mux.HandleFunc("PATCH "+api+"/users/{subject}", d.UserUpdate.Serve)
	// The instance-scoped grant routes live under the engine path, so a family mismatch is reported like any other connection error.
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/permissions", d.ListPermissions.Serve)
	mux.HandleFunc("PUT "+api+"/engine/{engine}/connections/{name}/permissions", d.SetPermission.Serve)
	mux.HandleFunc("DELETE "+api+"/engine/{engine}/connections/{name}/permissions", d.DeletePermission.Serve)

	// SQL-only: schema, row preview and the console ops.
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/schema", d.Schema.Serve)
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/erd", d.ERD.Serve)
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/tables/{table}/rows", d.Rows.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/query", d.Op.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/alter", d.Op.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/rollback", d.Op.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/analyze", d.Op.Serve)

	// Changes: SQL and MongoDB keep separate queues behind one route.
	mux.HandleFunc("GET "+api+"/engine/{engine}/changes", d.ChangeList.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/changes", d.ChangeSubmit.Serve)
	mux.HandleFunc("GET "+api+"/engine/{engine}/changes/{id}", d.ChangeGet.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/changes/{id}/approve", d.ChangeApprove.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/changes/{id}/apply", d.ChangeApply.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/changes/{id}/reject", d.ChangeReject.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/changes/{id}/schedule", d.ChangeSchedule.Serve)

	// Memory-cache-only: keys, values, removal and emptying the database.
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/keys", d.CacheKeys.Serve)
	mux.HandleFunc("DELETE "+api+"/engine/{engine}/connections/{name}/keys", d.CacheDel.Serve)
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/value", d.CacheVal.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/flush", d.CacheFlsh.Serve)

	// MongoDB-only: databases, collections and read requests.
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/databases", d.MongoDbs.Serve)
	mux.HandleFunc("GET "+api+"/engine/{engine}/connections/{name}/collections", d.MongoColls.Serve)
	mux.HandleFunc("POST "+api+"/engine/{engine}/connections/{name}/requests", d.MongoRun.Serve)

	mux.Handle("GET /assets/", http.HandlerFunc(getAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", getConsole)                     // the operator console (SPA shell)

	// Scheduling uses an authenticated external callback; no in-process ticker runs.
	mux.Handle("POST /internal/callback/scheduler", middlewares.WithSchedulerAuth(cfg.SchedulerKey, cfg.SchedulerSecret, http.HandlerFunc(d.Tick.Serve)))

	// Instances address a connection by its stable id, across families. The family routes live under {api}/engine/{engine}/..., so these paths never collide with the {engine} wildcard.
	mux.HandleFunc("GET "+api+"/instances", d.Instances.Serve)
	mux.HandleFunc("GET "+api+"/instances/{id}", d.Instance.Serve)
	mux.HandleFunc("POST "+api+"/instances/{id}/password", d.RotatePassword.Serve)

	return mux
}

func Run(ctx context.Context, log *slog.Logger, cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	store, err := sqlite.Open(ctx, cfg.DB, cfg.SecretKey)
	if err != nil {
		return err
	}
	defer store.Close()

	oidc, err := auth.New(ctx, cfg)
	if err != nil {
		return err
	}
	// pass a nil interface (not a typed-nil *auth.OIDC) so the service knows OIDC is off
	var verifier interfaces.Verifier
	if oidc != nil {
		verifier = oidc
	}
	authSvc := auth.NewService(store, verifier)

	log.InfoContext(ctx, "noxis serving", "addr", cfg.Listen, "db", cfg.DB)

	d := newDeps(store, cfg, authSvc)
	if err := d.Modules.Auth().EnsureRoot(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return err
	}

	// Middleware order is CORS → trace → auth → request log, so denied tokens are traced and actor fields attach after auth.
	handler := middlewares.CORS(middlewares.Trace(middlewares.WithAuth(authSvc, middlewares.RequestLog(newHandler(d, cfg), log), log)))
	srv := &http.Server{Addr: cfg.Listen, Handler: handler}
	baseCtx := common.WithModuleLoader(ctx, d.Modules)
	// every request ctx descends from ours
	srv.BaseContext = func(net.Listener) context.Context {
		return baseCtx
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		log.InfoContext(ctx, "noxis stopped")
		return nil
	}
}
