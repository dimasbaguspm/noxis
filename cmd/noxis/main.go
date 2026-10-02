package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"noxis/internal/common/config"
	"noxis/internal/common/logging"
)

// main is the only place wiring happens: env -> config struct -> logger ->
// app container -> one root command. Nothing else reads env or logs unbooted.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	log, shutdown, err := logging.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logging:", err)
		os.Exit(1)
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	if err := NewRootCmd(&App{Cfg: cfg, Log: log}).ExecuteContext(ctx); err != nil {
		log.Error("command failed", "error", err)
		os.Exit(1)
	}
}