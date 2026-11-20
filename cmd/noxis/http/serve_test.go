package http_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"strings"
	"testing"
	"time"

	apiserver "noxis/cmd/noxis/http"
)

// The serve command is a flag set and a listener; the handler is somebody else's. These tests pin
// exactly that split: what the flags resolve to, what a failed build does, and that a cancelled
// context is a clean exit rather than an error.

func TestServeFlagsAreTheConstants(t *testing.T) {
	cmd := apiserver.NewServe(func(context.Context, config.Config, string, *slog.Logger) (http.Handler, io.Closer, error) {
		t.Fatal("the flags were not resolved before the build ran")
		return nil, nil, nil
	})
	if got := cmd.Flags().Lookup("addr").DefValue; got != constants.ListenAddr {
		t.Fatalf("addr default = %q, want %q", got, constants.ListenAddr)
	}
	if got := cmd.Flags().Lookup("state").DefValue; got != constants.StatePath {
		t.Fatalf("state default = %q, want %q", got, constants.StatePath)
	}
	if cmd.Annotations[constants.AnnoNoToken] != "true" {
		t.Fatal("serve would require the session it is about to check")
	}
}

// A build that fails is the server's answer, verbatim: it is the state file or the key, and neither
// is something this package can fix.
func TestServeReportsABuildFailure(t *testing.T) {
	want := errors.New("no state file")
	cmd := apiserver.NewServe(func(context.Context, config.Config, string, *slog.Logger) (http.Handler, io.Closer, error) {
		return nil, nil, want
	})
	cmd.SetArgs([]string{"--addr", "127.0.0.1:0", "--state", t.TempDir() + "/state.db"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	t.Setenv("TOKENS", `{"ops":{"secret":"0123456789abcdef0123456789abcdef","scopes":["read"]}}`)

	if err := cmd.Execute(); !errors.Is(err, want) {
		t.Fatalf("err = %v, want the build failure", err)
	}
}

// The listener is the only thing that can fail after the build, and a cancelled context has to be an
// exit code of zero: a container that stops on SIGTERM is not a container that crashed.
func TestServeStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	built := make(chan struct{})
	cmd := apiserver.NewServe(func(context.Context, config.Config, string, *slog.Logger) (http.Handler, io.Closer, error) {
		close(built)
		return http.NewServeMux(), io.NopCloser(strings.NewReader("")), nil
	})
	cmd.SetArgs([]string{"--addr", "127.0.0.1:0"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	t.Setenv("TOKENS", `{"ops":{"secret":"0123456789abcdef0123456789abcdef","scopes":["read"]}}`)

	done := make(chan error, 1)
	go func() { done <- cmd.ExecuteContext(ctx) }()

	select {
	case <-built:
	case <-time.After(5 * time.Second):
		t.Fatal("the builder never ran")
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shutdown returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop on a cancelled context")
	}
}

// The healthcheck is the image's probe: it passes on the status the server answers with and fails on
// anything else, including a body that says the process is up while the API is not.
func TestHealthcheck(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"healthy", http.StatusOK, `{"data":{"status":"ok"}}`, ""},
		{"degraded", http.StatusOK, `{"data":{"status":"down"}}`, "not healthy"},
		{"unreachable", http.StatusServiceUnavailable, `{"error":{"code":"unavailable"}}`, "not healthy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			// The group is what a person runs, so the group is what the test runs: a subcommand
			// executed on its own does not inherit the flags the parent would have parsed.
			group := apiserver.NewHTTP(nil)
			group.SetArgs([]string{"healthcheck", "--url", server.URL})
			group.SetOut(io.Discard)
			group.SetErr(io.Discard)

			err := group.Execute()
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("healthy probe returned %v", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}
