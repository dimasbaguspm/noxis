package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"noxis/internal/client"
	"noxis/internal/common/config"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"time"

	"github.com/spf13/cobra"
)

// newHTTP is the server half of the binary: the process that owns the state file and answers the
// API the CLI talks to. It is a command, not a second executable, so an image is one file.
func newHTTP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Run the noXis API",
	}
	cmd.AddCommand(newHTTPServe(), newHTTPHealthcheck())
	return cmd
}

func newHTTPServe() *cobra.Command {
	var addr, statePath string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the REST API and own the state file",
		Args:  cobra.NoArgs,
		// The server is the thing that checks tokens; it cannot require one to start.
		Annotations: map[string]string{annoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serveAPI(cmd.Context(), addr, statePath)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", constants.ListenAddr,
		"listen address (a constant by default; a flag for development and the acceptance suite)")
	cmd.Flags().StringVar(&statePath, "state", constants.StatePath,
		"state file (a constant by default; a flag for development and the acceptance suite)")
	return cmd
}

// newHTTPHealthcheck is the image's HEALTHCHECK: the binary asks its own route, so the container
// needs no curl and the probe measures exactly what a client would get.
func newHTTPHealthcheck() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:         "healthcheck",
		Short:       "Ask the API whether it is alive",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{annoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := client.New(url, "")
			if err != nil {
				return err
			}
			health, err := api.Live(cmd.Context())
			if err != nil {
				return fmt.Errorf("not healthy: %w", err)
			}
			if health.Status != "ok" {
				return fmt.Errorf("not healthy: status %q", health.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "http://127.0.0.1"+constants.ListenAddr, "the API to ask")
	return cmd
}

// serveAPI reads the environment once, builds the transport once, and serves until the context is
// cancelled. Everything it needs is a constant or a flag, so the container needs no entrypoint
// script.
func serveAPI(ctx context.Context, addr, statePath string) error {
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

	handler, closer, err := buildServer(ctx, cfg, statePath, log)
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
