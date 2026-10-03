// Package server is the noxis daemon: REST API + embedded operator console
// over the connection registry and the SQL module. Run is the module entry.
package server

import (
	"context"
	"log/slog"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/repositories/sqlite"
)

// Run blocks serving REST until ctx is done or a dependency fails.
func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	if err := cfg.Require(entity.EnvListen, entity.EnvDB); err != nil {
		return err
	}
	store, err := sqlite.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer store.Close()

	log.InfoContext(ctx, "noxis serving", "addr", cfg.Listen, "db", cfg.DB)

	srv := &http.Server{Addr: cfg.Listen, Handler: newHandler(store, log)}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.InfoContext(ctx, "noxis stopped")
		return nil
	}
}
