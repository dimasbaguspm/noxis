package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"noxis/cmd/noxis/commands"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
)

// main is the only place wiring happens: env -> config struct -> logger ->
// app container -> one root command. Nothing else reads env or logs unbooted.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := entity.LoadConfig()
	log, shutdown, err := logging.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logging:", err)
		os.Exit(1)
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	if err := commands.NewRootCmd(&entity.App{Cfg: cfg, Log: log}).ExecuteContext(ctx); err != nil {
		log.ErrorContext(ctx, "command failed", "error", err)
		os.Exit(1)
	}
}
