package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	"noxis/internal/features/doorman"
)

func newSyncCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "sync NODE",
		Short: "Manually reconcile a node against its desired state (fallback)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			node := args[0]
			var spec struct {
				Engines []entity.Engine `json:"engines"`
			}
			if err := doJSON(cmd.Context(), app, "GET", "/nodes/"+node, nil, &spec); err != nil {
				return err
			}
			detail, err := doorman.NewReconciler(node).Reconcile(cmd.Context(), spec.Engines)
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
