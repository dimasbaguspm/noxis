package commands

import (
	"context"
	"net"
	"net/http"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"noxis/internal/entity"
	"noxis/internal/repository"
	rpc "noxis/internal/repository/grpc"
	"noxis/internal/repository/rest"
)

func newMasterCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "master",
		Short: "Run as master (REST API + internal slave transport)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if err := app.Cfg.Require(entity.EnvListen, entity.EnvGRPC, entity.EnvDB); err != nil {
				return err
			}
			repo, err := repository.OpenSQLite(app.Cfg.DB)
			if err != nil {
				return err
			}
			defer repo.Close()

			ctrl := rpc.NewServer(repo, repo)
			errs := make(chan error, 2)
			go serveREST(ctx, app, ctrl, errs)
			go serveGRPC(ctx, app, ctrl, errs)

			select {
			case err := <-errs:
				return err
			case <-ctx.Done():
				app.Log.InfoContext(ctx, "master stopped")
				return nil
			}
		},
	}
}

func serveGRPC(ctx context.Context, app *entity.App, ctrl *rpc.Server, errs chan<- error) {
	lis, err := net.Listen("tcp", app.Cfg.GRPC)
	if err != nil {
		errs <- err
		return
	}
	srv := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	rpc.RegisterControlServer(srv, ctrl)
	app.Log.InfoContext(ctx, "grpc transport serving", "addr", app.Cfg.GRPC)
	errs <- srv.Serve(lis)
}

func serveREST(ctx context.Context, app *entity.App, ctrl *rpc.Server, errs chan<- error) {
	app.Log.InfoContext(ctx, "rest api serving", "addr", app.Cfg.Listen)
	srv := &http.Server{Addr: app.Cfg.Listen, Handler: rest.New(ctrl)}
	errs <- srv.ListenAndServe()
}
