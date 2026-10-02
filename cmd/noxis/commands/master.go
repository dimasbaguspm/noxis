package commands

import (
	"net"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"noxis/internal/entity"
	"noxis/internal/repository"
	rpc "noxis/internal/repository/grpc"
)

func newMasterCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "master",
		Short: "Run as master (durable desired state + control plane)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if err := app.Cfg.Require(entity.EnvListen, entity.EnvDB); err != nil {
				return err
			}
			repo, err := repository.OpenSQLite(app.Cfg.DB)
			if err != nil {
				return err
			}
			defer repo.Close()

			lis, err := net.Listen("tcp", app.Cfg.Listen)
			if err != nil {
				return err
			}
			srv := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
			rpc.RegisterControlServer(srv, rpc.NewServer(repo, repo))

			app.Log.InfoContext(ctx, "master serving",
				"listen", app.Cfg.Listen, "db", app.Cfg.DB)
			serveErr := make(chan error, 1)
			go func() { serveErr <- srv.Serve(lis) }()
			select {
			case err := <-serveErr:
				return err
			case <-ctx.Done():
				srv.Stop() // immediate: long-lived Watch streams stall graceful drains
				app.Log.InfoContext(ctx, "master stopped")
				return nil
			}
		},
	}
}
