package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"noxis/internal/constants"
	"strings"
	"testing"
)

// One command, three depths: the argument count is the only thing that decides which question is asked, so
// the test asks all three and watches where each one lands.
func TestTableAnswersAtThreeDepths(t *testing.T) {
	const ref = "01a1fdf9-7535-7d2d-bb92-01a293970fe2"

	var asked string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Path
		w.Header().Set("Content-Type", constants.MediaTypeJSON)
		switch {
		case r.URL.Path == "/v1/targets/"+ref+"/databases":
			_, _ = w.Write([]byte(`{"data":["app"]}`))
		case strings.HasSuffix(r.URL.Path, "/users"):
			// the detail route answers an object: columns, indexes, DDL
			_, _ = w.Write([]byte(`{"data":{"columns":[],"indexes":[],"create_sql":"CREATE TABLE users ()"}}`))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	defer server.Close()

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"schemas", []string{ref}, "/v1/targets/" + ref + "/databases"},
		{"tables", []string{ref, "app"}, "/v1/targets/" + ref + "/databases/app/tables"},
		{"one table", []string{ref, "app", "users"}, "/v1/targets/" + ref + "/databases/app/tables/users"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asked = ""
			cmd := NewTable(&Session{APIURL: server.URL, Token: "t"})
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("table %v: %v", tc.args, err)
			}
			if asked != tc.want {
				t.Fatalf("table %v asked %q, want %q", tc.args, asked, tc.want)
			}
		})
	}
}

// The command's own page has to teach the three depths: it is the only place they are written down.
func TestTableHelpNamesTheDepths(t *testing.T) {
	cmd := NewTable(&Session{})
	for _, want := range []string{"table <target>", "table <target> <schema>", "table <target> <schema> <name>"} {
		if !strings.Contains(cmd.Long, want) {
			t.Fatalf("Long = %q, want it to name %q", cmd.Long, want)
		}
	}
}
