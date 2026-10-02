package main

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"noxis/internal/common/config"
	"noxis/internal/core"
)

// App is the bootstrap container wired once in main: config, logger, and
// everything derived from them. Commands receive *App and stay dumb.
type App struct {
	Cfg config.Config
	Log *slog.Logger
}

// NewRootCmd is the single place every command is scaffolded.
func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "noxis",
		Short: "Personal multi-machine infra control plane",
	}
	root.AddCommand(
		newMasterCmd(app),
		newSlaveCmd(app),
		newAssignCmd(app),
		newStatusCmd(app),
		newSyncCmd(app),
	)
	return root
}

// dial connects to the master configured via NOXIS_MASTER.
func (a *App) dial(ctx context.Context) (*grpc.ClientConn, error) {
	return core.Dial(ctx, a.Cfg.Master)
}