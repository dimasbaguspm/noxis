// Package master is the master daemon: REST API for frontends, events over
// Redpanda to slave workers.
package master

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"noxis/internal/control"
	"noxis/internal/entity"
	"noxis/internal/events"
	"noxis/internal/services"
)

// Run blocks serving REST until ctx is done or a dependency fails.
func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	if err := cfg.Require(entity.EnvListen, entity.EnvDB, entity.EnvBroker); err != nil {
		return err
	}
	svc, err := services.OpenSQLite(cfg.DB)
	if err != nil {
		return err
	}
	defer svc.DB.Close()

	k, err := events.Produce(ctx, strings.Split(cfg.Broker, ","))
	if err != nil {
		return err
	}
	defer k.Close()
	ctrl := control.New(svc.DB, k, log, strings.Split(cfg.Broker, ","))

	go func() {
		if err := ctrl.RunStatusConsumer(ctx); err != nil {
			log.ErrorContext(ctx, "status consumer stopped", "error", err)
		}
	}()

	log.InfoContext(ctx, "rest api serving", "addr", cfg.Listen, "broker", cfg.Broker)
	srv := &http.Server{Addr: cfg.Listen, Handler: newHandler(ctrl, log)}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.InfoContext(ctx, "master stopped")
		return nil
	}
}
