// Package server is the noxis daemon: REST API + embedded console.
package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"noxis/internal/entity"
	"noxis/internal/repositories/sqlite"
)

const shutdownTimeout = 10 * time.Second

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

	log.InfoContext(ctx, "noxis serving", "addr", cfg.Listen, "db", cfg.DB)

	deps := newDeps(store, log, cfg.TickKey)

	srv := &http.Server{Addr: cfg.Listen, Handler: newHandler(deps)}
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
