package rpc

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

	conn := bufconnConn(t, lis)
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

// TestQueryRoundTrip drives the full path: master Query RPC -> routed over a
// slave's live Watch stream -> slave answers -> caller gets the result.
func TestQueryRoundTrip(t *testing.T) {
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

	// slave side: open Watch, say hello, get the initial spec push
	slaveConn := bufconnConn(t, lis)
	defer slaveConn.Close()
	watch, err := NewControlClient(slaveConn).Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := watch.Send(&StreamMsg{Node: "themis"}); err != nil {
		t.Fatal(err)
	}
	msg, err := watch.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if msg.GetSpec() == nil {
		t.Fatalf("expected spec snapshot on connect, got %T", msg.Payload)
	}

	// caller side: query must travel the stream and come back
	queryRes := make(chan *QueryResult, 1)
	queryErr := make(chan error, 1)
	go func() {
		caller := bufconnConn(t, lis)
		defer caller.Close()
		res, err := NewControlClient(caller).Query(context.Background(),
			&QueryRequest{Node: "themis", Engine: "pg", Sql: "SELECT 1"})
		if err != nil {
			queryErr <- err
			return
		}
		queryRes <- res
	}()

	// slave answers the routed query
	msg, err = watch.Recv()
	if err != nil {
		t.Fatal(err)
	}
	q := msg.GetQuery()
	if q == nil || q.Sql != "SELECT 1" {
		t.Fatalf("expected routed query, got %T", msg.Payload)
	}
	if err := watch.Send(&StreamMsg{Payload: &StreamMsg_Result{Result: &QueryResult{Id: q.Id, Output: "42"}}}); err != nil {
		t.Fatal(err)
	}

	select {
	case res := <-queryRes:
		if res.Output != "42" {
			t.Fatalf("output = %q, want 42", res.Output)
		}
	case err := <-queryErr:
		t.Fatal(err)
	}
}

func bufconnConn(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}
