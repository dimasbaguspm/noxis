// Package main is the noxis CLI entry: `run master`, `run slave`, and
// `register slave` (env → config → logging → root command).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"noxis/cmd/noxis/master"
	slavecmd "noxis/cmd/noxis/slave/commands"
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
		slavecmd.NewRunCmd(log),
	)

	reg := &cobra.Command{
		Use:          "register",
		Short:        "One-shot registration",
		SilenceUsage: true}
	reg.AddCommand(slavecmd.NewRegisterCmd(log, cfg))

	root := &cobra.Command{
		Use:          "noxis",
		Short:        "Personal multi-machine infra control plane",
		SilenceUsage: true,
	}
	root.AddCommand(run, reg)

	return root.ExecuteContext(ctx)
}
