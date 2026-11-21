package http_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	apiserver "noxis/cmd/noxis/http"
	"noxis/cmd/noxis/http/handlers"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/common/config"
	"noxis/internal/constants"
	"strings"
	"testing"
)

// consoleStub stands where the embedded console would: it answers the shell for anything, because the
// question here is which paths reach it at all, not what it renders.
func consoleStub() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(constants.HeaderContentType, "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<!doctype html><title>the console</title>")
	})
}

func muxWithConsole(t *testing.T) http.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth, err := middlewares.NewAuthenticator([]config.Token{{Name: "ops", Secret: "secret", Scopes: []string{constants.ScopeRead}}}, log)
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}
	return apiserver.NewMux(apiserver.Deps{
		Log:      log,
		Auth:     auth,
		Handlers: &handlers.Handlers{},
		Console:  consoleStub(),
	})
}

// One process and one URL: the page the browser opens and the calls it makes arrive at the same mux,
// so an operator has one address to publish and no browser is ever asked about CORS.
func TestTheConsoleAnswersTheAPIsOwnOrigin(t *testing.T) {
	handler := muxWithConsole(t)
	for _, path := range []string{"/", "/targets/01a1/target", "/targets/01a1/target/proposals/01a2/change"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want the console", path, rec.Code)
		}
		if got := rec.Header().Get(constants.HeaderContentType); !strings.HasPrefix(got, "text/html") {
			t.Fatalf("GET %s content type = %q, want html", path, got)
		}
	}
}

// The console is the last route on the mux, so a typo in an API path must still be answered the way an
// API client can read it: an envelope with a code, not a page with a title.
func TestTheAPIsOwnNamespaceIsNeverTheConsole(t *testing.T) {
	handler := muxWithConsole(t)
	for _, path := range []string{"/v1/targts", "/v1/targets/01a1/nope", "/v1", "/openapi.json/nope"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", path, rec.Code)
		}
		if got := rec.Header().Get(constants.HeaderContentType); !strings.Contains(got, "json") {
			t.Fatalf("GET %s content type = %q, want json", path, got)
		}
		var answer respond.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if answer.Error == nil || answer.Error.Code != constants.CodeNotFound {
			t.Fatalf("GET %s error = %+v, want code %q", path, answer.Error, constants.CodeNotFound)
		}
		if !strings.Contains(answer.Error.Message, "console is at /") {
			t.Fatalf("GET %s message = %q, want it to say where the console is", path, answer.Error.Message)
		}
	}
}

// A binary with no console to serve keeps the API alone, and the mux must not grow a catch-all for it:
// an API-only build answering HTML on an unknown path would be the same bug in reverse.
func TestAnAPIWithoutAConsoleHasNoCatchAll(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth, err := middlewares.NewAuthenticator([]config.Token{{Name: "ops", Secret: "secret", Scopes: []string{constants.ScopeRead}}}, log)
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}
	handler := apiserver.NewMux(apiserver.Deps{Log: log, Auth: auth, Handlers: &handlers.Handlers{}})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET / with no console = %d, want 404", rec.Code)
	}
}
