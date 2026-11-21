package commands

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The escape hatch answers the envelope as it came, and asks the route it was given: that is the whole
// contract, and the reason a person reaches for it when the CLI has no verb yet.
func TestAPIGetPrintsTheEnvelope(t *testing.T) {
	var asked string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.Method + " " + r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"status":"ok"}}`))
	}))
	defer server.Close()

	out := &bytes.Buffer{}
	cmd := newAPICall(&Session{APIURL: server.URL, Token: "t"}, http.MethodGet)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"/v1/health?x=1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("api get: %v", err)
	}
	if asked != "GET /v1/health?x=1" {
		t.Fatalf("asked %q, want the path it was given", asked)
	}
	if !strings.Contains(out.String(), `"status": "ok"`) {
		t.Fatalf("output = %q, want the envelope, indented", out.String())
	}
}

// A write needs --yes: the command cannot tell a probe from a destructive route, so it does not try.
func TestAPIPostRefusesWithoutYes(t *testing.T) {
	cmd := newAPICall(&Session{APIURL: "http://127.0.0.1:1", Token: "t"}, http.MethodPost)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"/v1/targets/x/probe"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v, want a refusal naming --yes", err)
	}
}

// A path that names a host would be a second destination, and this command has exactly one.
func TestAPIRefusesAnAbsoluteURL(t *testing.T) {
	for _, path := range []string{"http://evil.example/v1/version", "v1/version"} {
		cmd := newAPICall(&Session{APIURL: "http://127.0.0.1:1", Token: "t"}, http.MethodGet)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{path})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "route") {
			t.Fatalf("path %q: err = %v, want a refusal", path, err)
		}
	}
}
