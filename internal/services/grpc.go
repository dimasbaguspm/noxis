package services

import (
	"context"

	"google.golang.org/grpc"

	rpc "noxis/internal/services/grpc"
)

// DialGRPC connects to the master's internal transport (used by the slave
// daemon on every reconnect attempt).
func DialGRPC(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return rpc.Dial(ctx, addr)
}
