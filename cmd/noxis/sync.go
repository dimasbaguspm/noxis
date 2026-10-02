package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"noxis/internal/common/config"
	"noxis/internal/core"
	"noxis/internal/features/doorman"
)

func newSyncCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "sync NODE",
		Short: "Manually reconcile a node against its desired state (fallback)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(config.EnvMaster); err != nil {
				return err
			}
			node := args[0]
			conn, err := app.dial(cmd.Context())
			if err != nil {
				return err
			}
			defer conn.Close()
			spec, err := core.NewControlClient(conn).GetDesiredState(cmd.Context(), &core.NodeId{Id: node})
			if err != nil {
				return err
			}
			detail, err := doorman.NewReconciler(node).Reconcile(cmd.Context(), core.SpecFromProto(spec).Engines)
			if err != nil {
				return err
			}
			if detail == "" {
				detail = "in sync"
			}
			fmt.Println(detail)
			return nil
		},
	}
}