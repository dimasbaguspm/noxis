package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
)

func newQueryCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "query NODE ENGINE \"SQL\"",
		Short: "Run a query against an engine on a node",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			var out struct {
				Output string `json:"output"`
			}
			if err := doJSON(cmd.Context(), app, "POST", "/nodes/"+args[0]+"/query",
				map[string]string{"engine": args[1], "sql": args[2]}, &out); err != nil {
				return err
			}
			fmt.Print(out.Output)
			if !strings.HasSuffix(out.Output, "\n") {
				fmt.Println()
			}
			return nil
		},
	}
}
