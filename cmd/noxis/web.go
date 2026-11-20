package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"noxis/internal/common/dotconfig"
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
		port    int
		apiURL  string
		profile string
	)
	cmd := &cobra.Command{
		Use:   "serve --api https://api.example.com",
		Short: "Serve the console from this binary",
		Args:  cobra.NoArgs,
		// The console is a page: it needs no token, and it never touches the state file.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serveConsole(cmd.Context(), cmd, port, apiURL, profile)
		},
	}
	cmd.Flags().IntVar(&port, "port", constants.ConsolePort, "the port the console listens on")
	cmd.Flags().StringVar(&apiURL, "api", "",
		"the API's public URL, as the browser reaches it (default: the API the CLI is signed in to)")
	cmd.Flags().StringVar(&profile, "profile", "",
		"which stored CLI profile to read the API and the token from (default the current one)")
	return cmd
}

// serveConsole serves one embedded build until the context is cancelled. It has no CORS and no API
// of its own: the console is a page, and the API is the process next to it.
func serveConsole(ctx context.Context, cmd *cobra.Command, port int, apiURL, profile string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is outside 1-65535", port)
	}
	// A signed-in CLI already knows which API it talks to and which token it uses: the console is the
	// same operator, in a browser, so it starts from the same session instead of asking again.
	session, err := cliSession(profile, profile != "")
	if err != nil {
		return err
	}
	if apiURL == "" {
		apiURL = session.APIURL
	}
	// The console is a page in somebody's browser: the address it calls must be the one the browser
	// can reach, not the one this process happens to see.
	api, err := web.ParseAPI(apiURL)
	if err != nil {
		return err
	}
	console, err := web.New(web.Config{API: api, Token: session.Token})
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
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "console on :%d, calling %s (%s, %s)\n",
		port, console.API(), built(console.Built()), signedIn(console.SignedIn()))

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

// cliSession is the CLI's stored profile, read here so the browser starts where the terminal is. A
// machine that never signed in is not an error: the console asks for a token, which is what it did
// before there was a session to inherit.
func cliSession(name string, explicit bool) (dotconfig.Profile, error) {
	cfg, err := dotconfig.Load()
	if err != nil {
		return dotconfig.Profile{}, err
	}
	if name == "" {
		name = cfg.Current
	}
	if name == "" {
		return dotconfig.Profile{}, nil
	}
	profile, err := cfg.Profile(name)
	if err != nil {
		// An explicitly named profile that is not there is a mistake; a machine that simply never
		// signed in is the console asking for a token, which is what it did before.
		if explicit {
			return dotconfig.Profile{}, fmt.Errorf("%w: %q is not in %s", dotconfig.ErrNoProfile, name, cfg.FilePath())
		}
		return dotconfig.Profile{}, nil
	}
	return profile, nil
}

func signedIn(session bool) string {
	if session {
		return "signed in: the browser inherits the CLI session"
	}
	return "no CLI session: the browser will ask for a token"
}

func built(compiled bool) string {
	if compiled {
		return "a console was built into this binary"
	}
	return "placeholder: run `npm run build` in web/ and copy it to internal/web/dist"
}
