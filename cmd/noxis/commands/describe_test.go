package commands

import (
	"strings"
	"testing"
)

// The document is the source of truth for what a route needs: the join is the whole command, so a
// fixture with a public route, a one-scope route and a two-scope route is the test.
const fixture = `{
  "info": {"version": "9.9.9"},
  "paths": {
    "/healthz": {"get": {"summary": "liveness", "tags": ["system"]}},
    "/v1/targets": {
      "get": {"summary": "the instances", "tags": ["targets"], "x-scopes": ["read"]},
      "post": {"summary": "register", "tags": ["targets"], "x-scopes": ["admin", "read"]},
      "parameters": [{"name": "limit"}]
    }
  }
}`

// A route the token cannot call is not hidden: an agent needs to know the door exists and why it is
// shut, so the operation is listed with allowed false rather than dropped.
func TestOperationsMarkWhatTheTokenMayCall(t *testing.T) {
	operations, version, err := operationsOf([]byte(fixture), []string{"read"})
	if err != nil {
		t.Fatalf("operations: %v", err)
	}
	if version != "9.9.9" {
		t.Fatalf("version = %q, want the document's own", version)
	}
	if len(operations) != 3 {
		t.Fatalf("operations = %d, want 3: `parameters` is not an operation", len(operations))
	}

	want := map[string]bool{
		"GET /healthz":     true,
		"GET /v1/targets":  true,
		"POST /v1/targets": false,
	}
	for _, operation := range operations {
		key := operation.Method + " " + operation.Path
		allowed, known := want[key]
		if !known {
			t.Fatalf("unexpected operation %q", key)
		}
		if operation.Allowed != allowed {
			t.Fatalf("%s allowed = %v, want %v (scopes %v)", key, operation.Allowed, allowed, operation.Scopes)
		}
	}
}

// Every scope is needed, not any: a route that asks for admin and read is not callable by read alone.
func TestHoldsNeedsEveryScope(t *testing.T) {
	cases := []struct {
		name   string
		token  []string
		needed []string
		want   bool
	}{
		{"public", nil, nil, true},
		{"nothing at all", nil, []string{"read"}, false},
		{"one of two", []string{"admin"}, []string{"admin", "read"}, false},
		{"both", []string{"read", "admin"}, []string{"admin", "read"}, true},
		{"more than asked", []string{"read", "admin", "apply"}, []string{"admin", "read"}, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := holds(test.token, test.needed); got != test.want {
				t.Fatalf("holds(%v, %v) = %v, want %v", test.token, test.needed, got, test.want)
			}
		})
	}
}

// A document that does not parse is refused rather than described as empty: an agent that gets an
// empty operation list would conclude it may do nothing.
func TestOperationsRefuseADocumentThatDoesNotParse(t *testing.T) {
	_, _, err := operationsOf([]byte("not json"), nil)
	if err == nil {
		t.Fatal("a broken document was described as if it were a contract")
	}
	if !strings.Contains(err.Error(), "does not parse") {
		t.Fatalf("error = %q", err)
	}
}
