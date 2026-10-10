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

	"noxis/cmd/noxis/server/api"
	"noxis/internal/modules"
	"noxis/internal/modules/auth"
	"noxis/internal/usecases"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/sqlite"
)

const shutdownTimeout = 10 * time.Second

func newVerifier(ctx context.Context, cfg entity.Config) (interfaces.Verifier, error) {
	oidc, err := auth.New(ctx, cfg)
	if err != nil || oidc == nil {
		return nil, err
	}
	return oidc, nil
}

// newApp wires the shared stores and the engine-family registry into the
// application use cases.
func newApp(store *sqlite.Store, log *slog.Logger) *usecases.App {
	return &usecases.App{
		Connections: store,
		Changes:     store,
		Grants:      store,
		Modules:     modules.Default(),
		Log:         log,
	}
}

// newServer wires the transport: the application plus the transport-only
// config (public OIDC info and the tick key).
func newServer(app *usecases.App, log *slog.Logger, cfg entity.Config) *api.Server {
	return &api.Server{
		App:     app,
		Log:     log,
		TickKey: cfg.TickKey,
		Auth: api.AuthInfo{
			Enabled:  cfg.OIDCIssuer != "",
			Issuer:   cfg.OIDCIssuer,
			ClientID: cfg.OIDCClientID,
		},
	}
}

// newHandler wires routes to handlers. The application is composed once and
// handlers carry it as a field, so no dependency rides the request context.
//
// Routes are generic over the engine family: `{engine}` names the family and
// every handler resolves its module from the registry. A family is registered
// exactly once, here, and a family that does not serve a route is refused by
// the handler (404) instead of reaching the wrong module.
func newHandler(apiSrv *api.Server, verifier interfaces.Verifier) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", api.GetHealthz)
	mux.HandleFunc("GET /v1/config/drivers", apiSrv.ListDrivers)
	mux.HandleFunc("GET /auth/config", apiSrv.GetAuthConfig)
	mux.HandleFunc("GET /v1/me", apiSrv.GetMe)

	// Connections: one generic handler per route, gated by the {engine} path
	// parameter through the module registry.
	mux.HandleFunc("GET /v1/engine/{engine}/connections", apiSrv.ListConnections)
	mux.HandleFunc("POST /v1/engine/{engine}/connections", apiSrv.SaveConnection)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}", apiSrv.GetConnection)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}", apiSrv.DeleteConnection)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/test", apiSrv.TestConnection)

	// Grants: per-connection RBAC policy, managed by global admins only.
	mux.HandleFunc("GET /v1/config/groups", apiSrv.ListConfigGroups)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/grants", apiSrv.ListGrants)
	mux.HandleFunc("PUT /v1/engine/{engine}/connections/{name}/grants", apiSrv.SetGrant)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/grants", apiSrv.DeleteGrant)

	// SQL-only: schema, row preview and the console ops.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/schema", apiSrv.GetSchema)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/tables/{table}/rows", apiSrv.GetRows)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/query", apiSrv.PostQuery)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/alter", apiSrv.PostAlter)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/rollback", apiSrv.PostRollback)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/analyze", apiSrv.PostAnalyze)

	// Changes: SQL and MongoDB keep separate queues behind one route.
	mux.HandleFunc("GET /v1/engine/{engine}/changes", apiSrv.ListChanges)
	mux.HandleFunc("POST /v1/engine/{engine}/changes", apiSrv.SubmitChange)
	mux.HandleFunc("GET /v1/engine/{engine}/changes/{id}", apiSrv.GetChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/approve", apiSrv.ApproveChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/apply", apiSrv.ApplyChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/reject", apiSrv.RejectChange)
	mux.HandleFunc("POST /v1/engine/{engine}/changes/{id}/schedule", apiSrv.ScheduleChange)

	// Memory-cache-only: keys, values, removal and emptying the database.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/keys", apiSrv.ListKeys)
	mux.HandleFunc("DELETE /v1/engine/{engine}/connections/{name}/keys", apiSrv.DeleteKeys)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/value", apiSrv.GetValue)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/flush", apiSrv.FlushDB)

	// MongoDB-only: databases, collections and read requests.
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/databases", apiSrv.ListDatabases)
	mux.HandleFunc("GET /v1/engine/{engine}/connections/{name}/collections", apiSrv.ListCollections)
	mux.HandleFunc("POST /v1/engine/{engine}/connections/{name}/requests", apiSrv.PostRequest)

	mux.Handle("GET /assets/", http.HandlerFunc(api.GetAssets)) // embedded FS keeps the assets/ prefix
	mux.HandleFunc("GET /", api.GetConsole)                     // the operator console (SPA shell)

	// Scheduling: an external cron (or the in-process scheduler) drives due
	// changes through the same handler.
	mux.HandleFunc("POST /internal/tick", apiSrv.PostTick)

	// Instances address a connection by its stable id, across families. The
	// family routes live under /v1/engine/{engine}/..., so these paths never
	// collide with the {engine} wildcard.
	mux.HandleFunc("GET /v1/instances", apiSrv.ListInstances)
	mux.HandleFunc("GET /v1/instances/{id}", apiSrv.GetInstance)
	mux.HandleFunc("POST /v1/instances/{id}/password", apiSrv.RotatePassword)

	return mux
}

func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	if err := cfg.Require(entity.EnvListen, entity.EnvDB); err != nil {
		return err
	}
	store, err := sqlite.Open(ctx, cfg.DB, cfg.SecretKey)
	if err != nil {
		return err
	}
	defer store.Close()
	if cfg.SecretKey == "" {
		log.WarnContext(ctx, "NOXIS_SECRET_KEY is not set; connection passwords are stored in plaintext")
	} else {
		log.InfoContext(ctx, "connection passwords are encrypted at rest")
	}

	verifier, err := newVerifier(ctx, cfg)
	if err != nil {
		return err
	}
	if verifier == nil {
		log.WarnContext(ctx, "NOXIS_OIDC_ISSUER is not set; the API is unauthenticated")
	} else {
		log.InfoContext(ctx, "OIDC authentication enabled", "issuer", cfg.OIDCIssuer, "adminGroup", cfg.OIDCAdminGroup)
	}

	log.InfoContext(ctx, "noxis serving", "addr", cfg.Listen, "db", cfg.DB)

	app := newApp(store, log)
	apiSrv := newServer(app, log, cfg)

	// authentication (when a verifier is configured), then CORS + request
	// logging around the route table
	handler := withMiddleware(withAuth(verifier, newHandler(apiSrv, verifier)), log)
	srv := &http.Server{Addr: cfg.Listen, Handler: handler}
	// every request ctx descends from ours
	srv.BaseContext = func(net.Listener) context.Context { return ctx }

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