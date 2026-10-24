package redismock

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func connectMock(t *testing.T, server *Server) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("127.0.0.1:%d", server.Port())})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return client
}

func TestServesScanAndGet(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Answer([]string{"greeting", "queue"}, "hello")
	client := connectMock(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keys, _, err := client.Scan(ctx, 0, "*", 10).Result()
	if err != nil || len(keys) != 2 {
		t.Fatalf("scan = %v, err = %v", keys, err)
	}
	if value, err := client.Get(ctx, "greeting").Result(); err != nil || value != "hello" {
		t.Fatalf("get = %q, err = %v", value, err)
	}
	commands := server.Commands()
	if len(commands) < 2 || !strings.EqualFold(commands[len(commands)-1], "GET greeting") {
		t.Fatalf("ledger = %v", commands)
	}
}

func TestServesError(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Fail("no such key")
	client := connectMock(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Get(ctx, "missing").Err(); err == nil {
		t.Fatal("expected a target error")
	}
}
