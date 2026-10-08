// Package server is the noxis daemon: REST API + embedded console.
package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/sqlite"
	"noxis/internal/services/auth"
)

const shutdownTimeout = 10 * time.Second

// newVerifier builds the token verifier, keeping a disabled issuer as an
// untyped nil interface: a typed-nil *auth.OIDC wrapped in interfaces.Verifier
// is not nil and would leave open mode silently enforcing auth.
func newVerifier(ctx context.Context, cfg entity.Config) (interfaces.Verifier, error) {
	oidc, err := auth.New(ctx, cfg)
	if err != nil || oidc == nil {
		return nil, err
	}
	return oidc, nil
}

// Run serves until ctx is done.
func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	if err := cfg.Require(entity.EnvListen, entity.EnvDB); err != nil {
		return err
	}
	store, err := sqlite.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer store.Close()

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

	deps := newDeps(store, log, cfg)

	srv := &http.Server{Addr: cfg.Listen, Handler: newHandler(deps, verifier)}
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
