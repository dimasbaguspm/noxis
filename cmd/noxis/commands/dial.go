package commands

import (
	"context"

	"google.golang.org/grpc"

	"noxis/internal/entity"
	rpc "noxis/internal/repository/grpc"
)

// dial connects to the master configured via NOXIS_MASTER.
func dial(ctx context.Context, app *entity.App) (*grpc.ClientConn, error) {
	return rpc.Dial(ctx, app.Cfg.Master)
}
