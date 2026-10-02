package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	rpc "noxis/internal/repository/grpc"
)

func newQueryCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "query NODE ENGINE \"SQL\"",
		Short: "Run a query against an engine on a node (routes over the watch stream)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			conn, err := dial(ctx, app)
			if err != nil {
				return err
			}
			defer conn.Close()
			res, err := rpc.NewControlClient(conn).Query(ctx, &rpc.QueryRequest{
				Node: args[0], Engine: args[1], Sql: args[2],
			})
			if err != nil {
				return err
			}
			if res.Error != "" {
				return fmt.Errorf("query error: %s", res.Error)
			}
			fmt.Print(res.Output) // raw client output, already human-shaped
			if !strings.HasSuffix(res.Output, "\n") {
				fmt.Println()
			}
			return nil
		},
	}
}
