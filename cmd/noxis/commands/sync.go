package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	"noxis/internal/features/doorman"
	rpc "noxis/internal/repository/grpc"
)

func newSyncCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "sync NODE",
		Short: "Manually reconcile a node against its desired state (fallback)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			node := args[0]
			conn, err := dial(ctx, app)
			if err != nil {
				return err
			}
			defer conn.Close()
			spec, err := rpc.NewControlClient(conn).GetDesiredState(ctx, &rpc.NodeId{Id: node})
			if err != nil {
				return err
			}
			detail, err := doorman.NewReconciler(node).Reconcile(ctx, rpc.SpecFromProto(spec).Engines)
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
