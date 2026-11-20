// Command noxisd is the noXis server: the REST API and the only process that writes the
// state file. It reads its environment once, builds the transport once, and then serves.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"noxis/internal/common/config"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
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
	addr := flag.String("addr", constants.ListenAddr, "listen address (a constant by default; a flag for development and the acceptance suite)")
	statePath := flag.String("state", constants.StatePath, "state file (a constant by default; a flag for development and the acceptance suite)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel, cfg.LogFormat)

	handler, closer, err := build(ctx, cfg, *statePath, log)
	if err != nil {
		return err
	}
	// The state file closes after the listener has stopped taking requests.
	defer func() { _ = closer.Close() }()

	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("noxisd listening", "addr", *addr, "state", *statePath)
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
