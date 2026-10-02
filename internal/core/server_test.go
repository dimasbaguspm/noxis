package core

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"noxis/internal/entity"
	"noxis/internal/repository"
)

// bufconn runs the real gRPC stack over an in-memory pipe: no listener, no
// ports, fast. Regression guard for the node-propagation bug: SetDesiredState
// must persist under the request's node so Watch/Get find it back.
func TestSetDesiredStateRoundTrip(t *testing.T) {
	repo, err := repository.OpenSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	RegisterControlServer(srv, NewServer(repo, repo))
	go srv.Serve(lis)
	defer srv.Stop()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := NewControlClient(conn)

	spec := &entity.Spec{Node: "themis", Engines: []entity.Engine{
		{Name: "redis", Type: "redis", Version: "7"},
	}}
	if _, err := client.SetDesiredState(context.Background(),
		&SetDesiredStateRequest{Node: "themis", Spec: SpecToProto(spec)}); err != nil {
		t.Fatalf("SetDesiredState: %v", err)
	}

	got, err := client.GetDesiredState(context.Background(), &NodeId{Id: "themis"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Engines) != 1 || got.Engines[0].Type != "redis" {
		t.Fatalf("GetDesiredState = %+v, want 1 redis engine", got)
	}

	// Default must not contaminate: another node has no spec.
	other, err := client.GetDesiredState(context.Background(), &NodeId{Id: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Engines) != 0 {
		t.Fatalf("other node got engines: %+v", other.Engines)
	}
}