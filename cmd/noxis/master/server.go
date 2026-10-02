// Package master is the master daemon: REST API for frontends plus the
// internal gRPC transport to slave workers.
package master

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"noxis/internal/entity"
	"noxis/internal/repository"
	rpc "noxis/internal/repository/grpc"
	"noxis/internal/services"
)

// Run blocks serving REST + gRPC until ctx is done or a listener fails.
func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	if err := cfg.Require(entity.EnvListen, entity.EnvGRPC, entity.EnvDB); err != nil {
		return err
	}
	svc, err := services.OpenSQLite(cfg.DB)
	if err != nil {
		return err
	}
	defer svc.DB.Close()

	ctrl := rpc.NewServer(svc.DB, svc.DB)
	errs := make(chan error, 2)
	go serveGRPC(ctx, log, cfg, ctrl, errs)
	go serveREST(ctx, log, cfg, ctrl, svc.DB, errs)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		log.InfoContext(ctx, "master stopped")
		return nil
	}
}

func serveGRPC(ctx context.Context, log *slog.Logger, cfg entity.Config, ctrl *rpc.Server, errs chan<- error) {
	lis, err := net.Listen("tcp", cfg.GRPC)
	if err != nil {
		errs <- err
		return
	}
	srv := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	rpc.RegisterControlServer(srv, ctrl)
	log.InfoContext(ctx, "grpc transport serving", "addr", cfg.GRPC)
	errs <- srv.Serve(lis)
}

func serveREST(ctx context.Context, log *slog.Logger, cfg entity.Config, ctrl *rpc.Server, repo *repository.SQLite, errs chan<- error) {
	log.InfoContext(ctx, "rest api serving", "addr", cfg.Listen)
	srv := &http.Server{Addr: cfg.Listen, Handler: newHandler(ctrl, repo, log)}
	errs <- srv.ListenAndServe()
}
