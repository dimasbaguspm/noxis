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
	"noxis/internal/auth"
	"noxis/internal/common"
	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/repositories"
	"noxis/internal/repositories/sqlite"
)

const shutdownTimeout = 10 * time.Second

// The store must satisfy every repository seam Default() fills; the assertion
// lives here because a package may not import its own sibling.
var _ repositories.Store = (*sqlite.Store)(nil)

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

	w := NewWiring(store, cfg, authSvc)
	if err := authSvc.EnsureRoot(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return err
	}

	// Middleware order is CORS → trace → auth → request log, so denied tokens are traced and actor fields attach after auth.
	handler := middlewares.CORS(middlewares.Trace(newHandler(w, cfg, authSvc, log)))
	srv := &http.Server{Addr: cfg.Listen, Handler: handler}
	baseCtx := common.WithModuleLoader(ctx, w.Modules)
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
