// Command noxisd is the noXis server: the REST API and the only process that writes the
// state file. It reads its environment once, builds the transport once, and then serves.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"noxis/internal/common/config"
	"noxis/internal/common/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel, cfg.LogFormat)

	handler, closer, err := build(ctx, cfg, log)
	if err != nil {
		return err
	}
	// The state file closes after the listener has stopped taking requests.
	defer func() { _ = closer.Close() }()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("noxisd listening", "addr", cfg.HTTPAddr, "state", cfg.StatePath)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
