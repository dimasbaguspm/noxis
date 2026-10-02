package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
)

type statusList struct {
	Nodes []struct {
		Node   string `json:"node"`
		State  string `json:"state"`
		Detail string `json:"detail"`
		Ts     int64  `json:"ts"`
	} `json:"nodes"`
}

func newStatusCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "status [NODE]",
		Short: "Show reported status of all nodes (or one)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			var list statusList
			if err := doJSON(cmd.Context(), app, "GET", "/status", nil, &list); err != nil {
				return err
			}
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			fmt.Printf("%-10s %-8s %-12s %s\n", "NODE", "STATE", "LAST SEEN", "DETAIL")
			for _, s := range list.Nodes {
				if filter != "" && s.Node != filter {
					continue
				}
				ago := "-"
				if t := time.Unix(s.Ts, 0); !t.IsZero() {
					ago = time.Since(t).Truncate(time.Second).String() + " ago"
				}
				fmt.Printf("%-10s %-8s %-12s %s\n", s.Node, s.State, ago, s.Detail)
			}
			return nil
		},
	}
}
