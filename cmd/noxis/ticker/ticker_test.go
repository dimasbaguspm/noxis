package ticker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/config"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPostsOnInterval(t *testing.T) {
	var calls atomic.Int32
	var key atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		key.Store(r.Header.Get("X-API-Key"))
		_, _ = io.WriteString(w, `{"applied":[],"failed":[]}`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := config.Config{TickHost: srv.URL, TickInterval: "10ms"}
	done := make(chan error, 1)
	go func() { done <- Run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), cfg) }()

	deadline := time.After(3 * time.Second)
	for calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("ticker made %d calls, want >= 2", calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if key.Load() != "" {
		t.Fatalf("ticker sent api key header = %q; key support was removed", key.Load())
	}
}

func TestRunRequiresURLAndKey(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := Run(context.Background(), log, config.Config{TickHost: "http://x", TickInterval: "nope"}); err == nil {
		t.Fatal("Run must reject an unparseable interval")
	}
	if err := Run(context.Background(), log, config.Config{}); err == nil {
		t.Fatal("Run must require the tick host")
	}
}
