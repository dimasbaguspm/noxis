package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/common/config"
	"noxis/internal/core"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status [NODE]",
		Short: "Show reported status of all nodes (or one)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(config.EnvMaster); err != nil {
				return err
			}
			conn, err := app.dial(cmd.Context())
			if err != nil {
				return err
			}
			defer conn.Close()
			list, err := core.NewControlClient(conn).ListStatus(cmd.Context(), &core.Empty{})
			if err != nil {
				return err
			}
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			fmt.Printf("%-10s %-8s %-12s %s\n", "NODE", "STATE", "LAST SEEN", "DETAIL")
			for _, s := range list.Statuses {
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