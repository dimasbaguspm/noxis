// Package main is the noxis CLI entry: `run server` (env → config →
// logging → root command).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"noxis/cmd/noxis/server"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// Bootstrap wires env -> config -> logger -> one root CLI (run/register).
func Bootstrap() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := entity.LoadConfig()
	log, shutdown, err := logging.New(cfg)

	if err != nil {
		return err
	}

	defer func() {
		_ = shutdown(ctx)
	}()

	run := &cobra.Command{
		Use:          "run",
		Short:        "Run a daemon",
		SilenceUsage: true}
	run.AddCommand(&cobra.Command{
		Use:   "server",
		Short: "Run the noxis server (REST API + console)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return server.Run(cmd.Context(), log, cfg)
		}})

	root := &cobra.Command{
		Use:          "noxis",
		Short:        "Lightweight SQL console: manage connections, run verified SQL",
		SilenceUsage: true,
	}
	root.AddCommand(run)

	return root.ExecuteContext(ctx)
}
