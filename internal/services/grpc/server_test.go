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

// TestQueryRoundTrip: RouteQuery -> slave's Watch stream -> answer comes
// back. Regression for stream routing.
func TestQueryRoundTrip(t *testing.T) {
	repo, err := repository.OpenSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	ctrl := NewServer(repo, repo)
	RegisterControlServer(srv, ctrl)
	go srv.Serve(lis)
	defer srv.Stop()

	// slave side: open Watch, hello, get the spec snapshot
	wconn := bufconnConn(t, lis)
	defer wconn.Close()
	watch, err := NewControlClient(wconn).Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := watch.Send(&StreamMsg{Node: "themis"}); err != nil {
		t.Fatal(err)
	}
	if msg, err := watch.Recv(); err != nil || msg.GetSpec() == nil {
		t.Fatalf("expected spec snapshot, got %v %v", msg, err)
	}

	// caller routes a query (plain method, what REST invokes)
	done := make(chan struct{})
	var got string
	var qerr error
	go func() {
		got, qerr = ctrl.RouteQuery(context.Background(), "themis", "pg", "SELECT 1")
		close(done)
	}()

	// slave answers on the stream
	msg, err := watch.Recv()
	if err != nil {
		t.Fatal(err)
	}
	q := msg.GetQuery()
	if q == nil || q.Sql != "SELECT 1" {
		t.Fatalf("expected routed query, got %+v", msg.Payload)
	}
	if err := watch.Send(&StreamMsg{Payload: &StreamMsg_Result{
		Result: &QueryResult{Id: q.Id, Output: "42"},
	}}); err != nil {
		t.Fatal(err)
	}

	<-done
	if qerr != nil {
		t.Fatal(qerr)
	}
	if got != "42" {
		t.Fatalf("got %q, want 42", got)
	}
}

// TestSpecPublish: SetNodeSpec notifies the stream, slave receives the new spec.
func TestSpecPublish(t *testing.T) {
	repo, err := repository.OpenSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	ctrl := NewServer(repo, repo)
	RegisterControlServer(srv, ctrl)
	go srv.Serve(lis)
	defer srv.Stop()

	wconn := bufconnConn(t, lis)
	defer wconn.Close()
	watch, err := NewControlClient(wconn).Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = watch.Send(&StreamMsg{Node: "themis"})
	if _, err := watch.Recv(); err != nil {
		t.Fatal(err)
	}

	if err := ctrl.SetNodeSpec(context.Background(), &entity.Spec{
		Node: "themis", Engines: []entity.Engine{{Name: "redis", Type: "redis", Version: "7"}},
	}); err != nil {
		t.Fatal(err)
	}

	msg, err := watch.Recv()
	if err != nil {
		t.Fatal(err)
	}
	spec := msg.GetSpec()
	if spec == nil || len(spec.Engines) != 1 || spec.Engines[0].Version != "7" {
		t.Fatalf("published spec = %+v", spec)
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
