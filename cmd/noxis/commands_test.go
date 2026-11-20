package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// stub is a noxisd that answers every route with the smallest valid envelope. The commands are the
// subject here, not the API: what matters is that each one builds its request, reads the answer and
// prints it.
func stub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := `{"data":{}}`
		switch {
		case r.URL.Path == "/v1/whoami":
			body = `{"data":{"name":"agent","scopes":["read","propose"]}}`
		case strings.HasSuffix(r.URL.Path, "/events"):
			body = `{"data":[]}`
		case strings.HasSuffix(r.URL.Path, "/databases"):
			body = `{"data":["app"]}`
		case strings.Contains(r.URL.Path, "/tables/"):
			body = `{"data":{"name":"users"}}`
		case strings.HasSuffix(r.URL.Path, "/tables"):
			body = `{"data":[{"name":"users"}]}`
		case strings.HasSuffix(r.URL.Path, "/query"):
			body = `{"data":{"columns":[{"name":"id"}],"values":[["1"]],"row_count":1}}`
		case strings.HasSuffix(r.URL.Path, "/capabilities"):
			body = `{"data":{"driver":"mysql","version":"8.4.2"}}`
		case r.Method == http.MethodGet && r.URL.Path == "/v1/targets":
			body = `{"data":[{"ref":"0192f0c0-0000-7000-8000-000000000001","name":"prod"}]}`
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proposals":
			body = `{"data":[{"ref":"0192f0c0-0000-7000-8000-000000000002","status":"verified"}]}`
		case r.URL.Path == "/v1/targets":
			body = `{"data":{"ref":"0192f0c0-0000-7000-8000-000000000001","name":"prod","has_password":true}}`
		case strings.HasSuffix(r.URL.Path, "/proposals"):
			body = `{"data":{"ref":"0192f0c0-0000-7000-8000-000000000002","status":"verified"}}`
		case strings.HasPrefix(r.URL.Path, "/v1/proposals/"):
			body = `{"data":{"ref":"0192f0c0-0000-7000-8000-000000000002","status":"approved"}}`
		case strings.HasPrefix(r.URL.Path, "/v1/targets/"):
			body = `{"data":{"ref":"0192f0c0-0000-7000-8000-000000000001","name":"prod"}}`
		}
		_, _ = w.Write([]byte(body))
	}))
}

const (
	stubTarget   = "0192f0c0-0000-7000-8000-000000000001"
	stubProposal = "0192f0c0-0000-7000-8000-000000000002"
)

// Every command, run once against the stub. A command that cannot build its request, read the
// envelope or print it fails here rather than in somebody's terminal.
func TestCommandsRun(t *testing.T) {
	isolated(t)
	server := stub(t)
	defer server.Close()

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"whoami", []string{"whoami"}, "agent"},
		{"whoami json", []string{"whoami", "--json"}, "agent"},
		{"version", []string{"version"}, "noxis"},
		{"target add", []string{
			"target", "add", "--name", "prod", "--host", "db", "--database", "app",
			"--username", "noxis", "--password", "hunter2",
		}, ""},
		{"target ls", []string{"target", "ls"}, "prod"},
		{"target show", []string{"target", "show", stubTarget}, ""},
		{"target probe", []string{"target", "probe", stubTarget}, ""},
		{"target ls json", []string{"target", "ls", "--json"}, "prod"},
		{"target archive", []string{"target", "archive", stubTarget}, "archived"},
		{"query", []string{"query", stubTarget, "--sql", "SELECT 1"}, ""},
		{"databases", []string{"databases", stubTarget}, "app"},
		{"tables", []string{"tables", stubTarget, "app"}, "users"},
		{"table", []string{"table", stubTarget, "app", "users"}, "COLUMN"},
		{"propose", []string{
			"propose", stubTarget, "--title", "add a column", "--sql",
			"ALTER TABLE users ADD COLUMN note TEXT",
		}, "verified"},
		{"proposals", []string{"proposals"}, "verified"},
		{"show-proposal", []string{"show-proposal", stubProposal}, "approved"},
		{"rehearse", []string{"rehearse", stubProposal}, "approved"},
		{"approve", []string{"approve", stubProposal, "--note", "fine"}, "approved"},
		{"reject", []string{"reject", stubProposal, "--note", "no"}, "approved"},
		{"apply", []string{"apply", stubProposal}, "approved"},
		{"apply destructive", []string{"apply", stubProposal, "--confirm-destructive"}, "approved"},
		{"events", []string{"events", stubProposal}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append(tc.args, "--api-url", server.URL, "--token", "test-token")
			root := newRoot()
			var out, errOut bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errOut)
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("%v: %v (stderr: %s)", args, err, errOut.String())
			}
			if tc.want != "" && !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q, want it to mention %q", out.String(), tc.want)
			}
		})
	}
}

// A statement can come from a file, which is how a long ALTER lands in a proposal without a shell
// quoting accident.
func TestQueryReadsStatementFromAFile(t *testing.T) {
	isolated(t)
	server := stub(t)
	defer server.Close()

	path := t.TempDir() + "/statement.sql"
	if err := os.WriteFile(path, []byte("SELECT id FROM users\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	root := newRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"query", stubTarget, "--file", path, "--api-url", server.URL, "--token", "t"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query --file: %v", err)
	}
}

// A refusal from the server reaches the terminal as the API described it, not as a status code.
func TestCommandReportsTheApiRefusal(t *testing.T) {
	isolated(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"proposal 0192 does not exist"}}`))
	}))
	defer server.Close()

	root := newRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"show-proposal", stubProposal, "--api-url", server.URL, "--token", "t"})
	err := root.Execute()
	if err == nil {
		t.Fatal("a 404 was reported as success")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("error = %q, want the server's message", err)
	}
}

// The list command prints the cursor when the server has another page.
func TestTargetListPrintsTheCursor(t *testing.T) {
	isolated(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"ref":"` + stubTarget + `","name":"prod"}],"meta":{"next_cursor":"` + stubTarget + `"}}`))
	}))
	defer server.Close()

	root := newRoot()
	var out, warnings bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&warnings)
	root.SetArgs([]string{"target", "ls", "--json", "--api-url", server.URL, "--token", "t"})
	if err := root.Execute(); err != nil {
		t.Fatalf("target ls: %v", err)
	}

	// `--json` prints the envelope the API sent, so an agent parses one shape whether it read the CLI or
	// the wire.
	var printed struct {
		Data []struct {
			Ref string `json:"ref"`
		} `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &printed); err != nil {
		t.Fatalf("output is not json: %v (%s)", err, out.String())
	}
	if len(printed.Data) == 0 || printed.Data[0].Ref == "" {
		t.Fatalf("the page carried no data: %s", out.String())
	}
	if printed.Meta.NextCursor == "" {
		t.Fatalf("the cursor was not printed: %s", out.String())
	}
}
