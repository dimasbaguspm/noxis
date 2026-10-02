package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"noxis/cmd/noxis/master"
	"noxis/cmd/noxis/slave"
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
	run.AddCommand(
		&cobra.Command{
			Use:   "master",
			Short: "Run the master daemon",
			RunE: func(cmd *cobra.Command, _ []string) error {
				return master.Run(cmd.Context(), log, cfg)
			}},
		slave.NewRunCmd(log),
	)

	reg := &cobra.Command{
		Use:          "register",
		Short:        "One-shot registration",
		SilenceUsage: true}
	reg.AddCommand(slave.NewRegisterCmd(log, cfg))

	root := &cobra.Command{
		Use:          "noxis",
		Short:        "Personal multi-machine infra control plane",
		SilenceUsage: true,
	}
	root.AddCommand(run, reg)

	return root.ExecuteContext(ctx)
}
