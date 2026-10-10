// Package server is the noxis REST API and operator console: the transport
// wiring. Swagger annotations live on the handlers in the api package.
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

	"noxis/cmd/noxis/server/deps"
	"noxis/cmd/noxis/server/handlers"
	"noxis/cmd/noxis/server/middlewares"
	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/modules/auth"
	cache "noxis/internal/repositories/in-memory-cache"
	"noxis/internal/repositories/sqlite"
)

const shutdownTimeout = 10 * time.Second

func newVerifier(ctx context.Context, cfg config.Config) (interfaces.Verifier, error) {
	oidc, err := auth.New(ctx, cfg)
	if err != nil || oidc == nil {
		return nil, err
	}
	return oidc, nil
}

// newDeps assembles the runtime container once: the stores, the engine
// registry and the transport values, then binds every use case to them.
func newDeps(store *sqlite.Store, log *slog.Logger, cfg config.Config) deps.Deps {
	return deps.Init(deps.Deps{
		Connections: store,
		Changes:     store,
		Grants:      store,
		Modules:     modules.Default(),
		Log:         log,
		Cache:       cache.New(),
		Auth: config.AuthInfo{
			Enabled:  cfg.OIDCIssuer != "",
			Issuer:   cfg.OIDCIssuer,
			ClientID: cfg.OIDCClientID,
		},
	})
}

// newHandler wires routes to plain handlers. Every request context carries
// the deps seeded in Run; a handler reads what it needs with usecases.From.
//
// Routes are generic over the engine family: `{engine}` names the family and
// every handler resolves its module from the registry. A family is registered
// exactly once, here, and a family that does not serve a route is refused by
// the handler (404) instead of reaching the wrong module.
func newHandler(verifier interfaces.Verifier) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.GetHealthz)
	mux.HandleFunc("GET /v1/config/drivers", handlers.ListDrivers)
	mux.HandleFunc("GET /auth/config", handlers.GetAuthConfig)
	mux.HandleFunc("GET /v1/me", handlers.GetMe)

	// Connections: one generic handler per route, gated by the {engine} path
	// parameter through the module registry.
	mux.HandleFunc("GET /v1/engine/{engine}/connections", handlers.ListConnections)
	mux.HandleFunc("POST /v1/engine/{engine}/connections", handlers.SaveConnection)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}", handlers.GetConnection)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}", handlers.DeleteConnection)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/test", handlers.TestConnection)

	// Grants: per-connection RBAC policy, managed by global admins only.
	mux.HandleFunc("GET /v1/config/groups", handlers.ListConfigGroups)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/grants", handlers.ListGrants)
	mux.HandleFunc("PUT /v1/engine/{engine}/connections/{name}/grants", handlers.SetGrant)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/grants", handlers.DeleteGrant)

	// SQL-only: schema, row preview and the console ops.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/schema", handlers.GetSchema)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/tables/{table}/rows", handlers.GetRows)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/query", handlers.PostQuery)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/alter", handlers.PostAlter)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/rollback", handlers.PostRollback)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/analyze", handlers.PostAnalyze)

	// Changes: SQL and MongoDB keep separate queues behind one route.
	mux.HandleFunc("GET /v1/engine/{engine}/changes", handlers.ListChanges)
	mux.HandleFunc("POST /v1/engine/{engine}/changes", handlers.SubmitChange)
	mux.HandleFunc("GET /v1/engine/{engine}/changes/{id}", handlers.GetChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/approve", handlers.ApproveChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/apply", handlers.ApplyChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/reject", handlers.RejectChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/schedule", handlers.ScheduleChange)

	// Memory-cache-only: keys, values, removal and emptying the database.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/keys", handlers.ListKeys)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/keys", handlers.DeleteKeys)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/value", handlers.GetValue)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/flush", handlers.FlushDB)

	// MongoDB-only: databases, collections and read requests.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/databases", handlers.ListDatabases)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/collections", handlers.ListCollections)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/requests", handlers.PostRequest)

	mux.Handle("GET /assets/", http.HandlerFunc(handlers.GetAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", handlers.GetConsole)                     // the operator console (SPA shell)

	// Scheduling: an external cron (or the in-process scheduler) drives due
	// changes through the same handler.
	mux.HandleFunc("POST /internal/callback/scheduler", handlers.PostScheduler)

	// Instances address a connection by its stable id, across families. The
	// family routes live under /v1/engine/{engine}/..., so these paths never
	// collide with the {engine} wildcard.
	mux.HandleFunc("GET /v1/instances", handlers.ListInstances)
	mux.HandleFunc("GET /v1/instances/{id}", handlers.GetInstance)
	mux.HandleFunc("POST /v1/instances/{id}/password", handlers.RotatePassword)

	return mux
}

func Run(ctx context.Context, log *slog.Logger, cfg config.Config) error {
	if err := cfg.Require(config.EnvListen, config.EnvDB); err != nil {
		return err
	}
	store, err := sqlite.Open(ctx, cfg.DB, cfg.SecretKey)
	if err != nil {
		return err
	}
	defer store.Close()

	verifier, err := newVerifier(ctx, cfg)
	if err != nil {
		return err
	}

	log.InfoContext(ctx, "noxis serving", "addr", cfg.Listen, "db", cfg.DB)

	d := newDeps(store, log, cfg)

	// CORS, then authentication (when a verifier is configured), then request
	// logging + span: the request log and its span run after the principal is
	// attached, so the user's trace attributes and log fields are set once,
	// in the middleware layer.
	handler := middlewares.CORS(middlewares.WithAuth(verifier, middlewares.RequestLog(newHandler(verifier), log)))
	srv := &http.Server{Addr: cfg.Listen, Handler: handler}
	// every request ctx descends from ours, seeded once with the deps
	srv.BaseContext = func(net.Listener) context.Context {
		return deps.Seed(modules.WithRegistry(ctx, d.Modules), d)
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
