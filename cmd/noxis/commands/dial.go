package commands

import (
	"context"

	"google.golang.org/grpc"

	rpc "noxis/internal/repository/grpc"
)

// dial connects to the master's internal gRPC transport (NOXIS_GRPC).
func dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return rpc.Dial(ctx, addr)
}
