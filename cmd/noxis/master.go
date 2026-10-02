package main

import (
	"net"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"noxis/internal/common/config"
	"noxis/internal/core"
	"noxis/internal/repository"
)

func newMasterCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "master",
		Short: "Run as master (durable desired state + control plane)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Cfg.Require(config.EnvListen, config.EnvDB); err != nil {
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
			core.RegisterControlServer(srv, core.NewServer(repo, repo))

			app.Log.Info("master serving",
				"listen", app.Cfg.Listen, "db", app.Cfg.DB)
			serveErr := make(chan error, 1)
			go func() { serveErr <- srv.Serve(lis) }()
			select {
			case err := <-serveErr:
				return err
			case <-cmd.Context().Done():
				srv.Stop() // immediate: long-lived Watch streams stall graceful drains
				app.Log.Info("master stopped")
				return nil
			}
		},
	}
}

