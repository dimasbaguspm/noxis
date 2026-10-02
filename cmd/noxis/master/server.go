// Package master is the master daemon: REST API for frontends, events over
// Kafka to slave workers.
package master

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"noxis/internal/control"
	"noxis/internal/entity"
	"noxis/internal/repositories/kafka"
	"noxis/internal/repositories/sqlite"
)

// Run blocks serving REST until ctx is done or a dependency fails.
func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	if err := cfg.Require(entity.EnvListen, entity.EnvDB, entity.EnvBroker, entity.EnvTopicPrefix); err != nil {
		return err
	}
	store, err := sqlite.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer store.Close()

	k, err := kafka.Produce(ctx, strings.Split(cfg.Broker, ","))
	if err != nil {
		return err
	}
	defer k.Close()
	ctrl := control.New(store, kafka.Topics{Prefix: cfg.TopicPrefix}, k, log, strings.Split(cfg.Broker, ","))

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
