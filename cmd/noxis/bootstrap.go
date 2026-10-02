package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"noxis/cmd/noxis/master"
	"noxis/cmd/noxis/slave"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// Bootstrap wires env -> config -> logger -> app -> one root command.
func Bootstrap() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, shutdown, err := setup(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = shutdown(ctx) }()
	if err := newRootCmd(app).ExecuteContext(ctx); err != nil {
		app.Log.ErrorContext(ctx, "command failed", "error", err)
		return err
	}
	return nil
}

func setup(ctx context.Context) (*entity.App, func(context.Context) error, error) {
	cfg := entity.LoadConfig()
	log, shutdown, err := logging.New(cfg)
	if err != nil {
		return nil, nil, err
	}
	return &entity.App{Cfg: cfg, Log: log}, shutdown, nil
}

func newRootCmd(app *entity.App) *cobra.Command {
	root := &cobra.Command{Use: "noxis", Short: "Personal multi-machine infra control plane"}
	run := &cobra.Command{Use: "run", Short: "Run a daemon"}
	run.AddCommand(
		&cobra.Command{Use: "master", Short: "Run the master daemon", RunE: func(cmd *cobra.Command, _ []string) error {
			return master.Run(cmd.Context(), app)
		}},
		slave.NewRunCmd(app),
	)
	reg := &cobra.Command{Use: "register", Short: "One-shot registration"}
	reg.AddCommand(slave.NewRegisterCmd(app))
	root.AddCommand(run, reg, newStatusCmd(app))
	return root
}

// newStatusCmd is the only read CLI verb; everything else lives in the REST
// API for frontends.
func newStatusCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "status [NODE]",
		Short: "Show node status (reported + registered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			var list struct {
				Nodes []struct {
					Node   string `json:"node"`
					State  string `json:"state"`
					Detail string `json:"detail"`
					Ts     int64  `json:"ts"`
				} `json:"nodes"`
			}
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
				if s.Ts > 0 {
					t := time.Unix(s.Ts, 0)
					ago = time.Since(t).Truncate(time.Second).String() + " ago"
				}
				fmt.Printf("%-10s %-8s %-12s %s\n", s.Node, s.State, ago, s.Detail)
			}
			return nil
		},
	}
}
