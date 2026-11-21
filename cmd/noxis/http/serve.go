package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"noxis/internal/client"
	"noxis/internal/common/config"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"time"

	"github.com/spf13/cobra"
)

// Builder is how the transport gets an API without naming an adapter: the composition root in
// package main builds the handler and this package only serves it. It is a parameter rather than an
// import because a transport that constructs a database is a transport nobody can test.
type Builder func(ctx context.Context, cfg config.Config, statePath string, log *slog.Logger) (http.Handler, io.Closer, error)

// The API's general information, read by `make spec`. Every operation is annotated where it is
// handled; this block is the part that is about the API rather than about one route.
//
// @title noXis API
// @version 2.0.0-dev
// @description Reviewed access to MySQL and MariaDB. An agent reads through `read`, proposes through `propose`, and a human holding `approve` and `apply` decides. A proposal is rehearsed — the schema is rebuilt in a throwaway database and the rollback must restore it — so the reviewer sees evidence rather than a check mark.
// @contact.name noXis
// @license.name MIT
// @host localhost:9000
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description `Bearer <token>`. The scopes a route needs are declared in the route table; a token without them is refused before the handler runs.
//
// NewHTTP is the server half of the binary: the process that owns the state file and answers the
// API the CLI talks to. It is a command, not a second executable, so an image is one file.
func NewHTTP(build Builder) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Run the noXis API",
	}
	cmd.AddCommand(NewServe(build), newHealthcheck())
	return cmd
}

func NewServe(build Builder) *cobra.Command {
	var addr, statePath string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the REST API and own the state file",
		Args:  cobra.NoArgs,
		// The server is the thing that checks tokens; it cannot require one to start.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serveAPI(cmd.Context(), build, addr, statePath)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", constants.ListenAddr,
		"listen address (a constant by default; a flag for development and the acceptance suite)")
	cmd.Flags().StringVar(&statePath, "state", constants.StatePath,
		"state file (a constant by default; a flag for development and the acceptance suite)")
	return cmd
}

// newHealthcheck is the image's HEALTHCHECK: the binary asks its own route, so the container
// needs no curl and the probe measures exactly what a client would get. One image serves two faces — the
// API on constants.ListenAddr, the console on constants.ConsolePort — and the image cannot know which one
// a container runs, so the default asks the API and then the console. Health means "the face this
// container runs answers", not "the port I assumed answers".
func newHealthcheck() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:         "healthcheck",
		Short:       "Ask the process in this container whether it is alive",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			targets := []string{url}
			if !cmd.Flags().Changed("url") {
				targets = append(targets, fmt.Sprintf("http://127.0.0.1:%d", constants.ConsolePort))
			}
			var last error
			for _, target := range targets {
				if err := alive(cmd.Context(), target); err != nil {
					last = err
					continue
				}
				return nil
			}
			return last
		},
	}
	cmd.Flags().StringVar(&url, "url", "http://127.0.0.1"+constants.ListenAddr,
		"the address to ask (default: the API, then the console)")
	return cmd
}

// alive asks one address for its own /healthz and insists on the status both faces answer with.
func alive(ctx context.Context, target string) error {
	api, err := client.New(target, "")
	if err != nil {
		return err
	}
	health, err := api.Live(ctx)
	if err != nil {
		return fmt.Errorf("not healthy: %w", err)
	}
	if health.Status != "ok" {
		return fmt.Errorf("not healthy: status %q", health.Status)
	}
	return nil
}

// serveAPI reads the environment once, builds the transport once, and serves until the context is
// cancelled. Everything it needs is a constant or a flag, so the container needs no entrypoint
// script.
func serveAPI(ctx context.Context, build Builder, addr, statePath string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, stopTelemetry, err := telemetry.New(ctx, cfg)
	if err != nil {
		return err
	}
	// Telemetry flushes last: a span that is still batched when the process exits is a span the
	// collector never sees.
	defer func() {
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), telemetry.FlushTimeout)
		defer cancel()
		_ = stopTelemetry(flush)
	}()

	handler, closer, err := build(ctx, cfg, statePath, log)
	if err != nil {
		return err
	}
	// The state file closes after the listener has stopped taking requests.
	defer func() { _ = closer.Close() }()

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("noxisd listening", "addr", addr, "state", statePath)
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
		if err := server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}
