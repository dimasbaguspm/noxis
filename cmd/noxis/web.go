package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"noxis/internal/constants"
	"noxis/internal/web"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// newWeb is the console half of the binary: the assets ship inside it, so a machine that has noxis
// has the console, and a machine that has not built one is told so instead of getting a 404.
func newWeb() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Run the noXis console",
	}
	cmd.AddCommand(newWebServe())
	return cmd
}

func newWebServe() *cobra.Command {
	var (
		port   int
		apiURL string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the console from this binary",
		Args:  cobra.NoArgs,
		// The console is a page: it needs no token, and it never touches the state file.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serveConsole(cmd.Context(), cmd, port, apiURL)
		},
	}
	cmd.Flags().IntVar(&port, "port", constants.ConsolePort, "the port the console listens on")
	cmd.Flags().StringVar(&apiURL, "api", constants.DefaultAPIURL, "the API the console is told to call")
	return cmd
}

// serveConsole serves one embedded build until the context is cancelled. It has no CORS and no API
// of its own: the console is a page, and the API is the process next to it.
func serveConsole(ctx context.Context, cmd *cobra.Command, port int, apiURL string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is outside 1-65535", port)
	}
	console, err := web.New(apiURL)
	if err != nil {
		return err
	}

	listen := net.ListenConfig{}
	listener, err := listen.Listen(ctx, "tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("console: listen on port %d: %w", port, err)
	}

	server := &http.Server{
		Handler:           console.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "console on http://localhost:%d, calling %s (%s)\n",
		port, console.API(), built(console.Built()))

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

func built(compiled bool) string {
	if compiled {
		return "a console was built into this binary"
	}
	return "placeholder: run `npm run build` in web/ and copy it to internal/web/dist"
}
