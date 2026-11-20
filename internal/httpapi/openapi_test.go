package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"noxis/internal/httpapi"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// document is as much of the contract as the test has to understand to keep it honest.
type document struct {
	OpenAPI  string                    `yaml:"openapi"`
	Paths    map[string]map[string]any `yaml:"paths"`
	Info     map[string]any            `yaml:"info"`
	Security []map[string]any          `yaml:"security"`
}

func load(t *testing.T) document {
	t.Helper()
	var doc document
	if err := yaml.Unmarshal(httpapiContract(t), &doc); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	return doc
}

// A route the document does not describe is a route nobody can call on purpose, and a scope that
// disagrees with the route table is worse: the document would promise a permission the server does
// not grant.
func TestEveryRouteIsDocumented(t *testing.T) {
	doc := load(t)
	if !strings.HasPrefix(doc.OpenAPI, "3.1") {
		t.Fatalf("openapi = %q, want 3.1.x", doc.OpenAPI)
	}
	if doc.Info["title"] == nil || doc.Info["version"] == nil {
		t.Fatal("info needs a title and a version")
	}

	for _, route := range httpapi.Routes(httpapi.Deps{}) {
		item, ok := doc.Paths[route.Path]
		if !ok {
			t.Fatalf("path %s is served but not documented", route.Path)
		}
		operation, ok := item[strings.ToLower(route.Method)].(map[string]any)
		if !ok {
			t.Fatalf("%s %s is served but not documented", route.Method, route.Path)
		}

		declared := scopeList(operation["x-scopes"])
		want := append([]string(nil), route.Scopes...)
		sort.Strings(declared)
		sort.Strings(want)
		if strings.Join(declared, ",") != strings.Join(want, ",") {
			t.Fatalf("%s %s declares scopes %v, the route table says %v", route.Method, route.Path, declared, want)
		}
		if route.Public {
			if _, ok := operation["security"]; !ok {
				t.Fatalf("%s %s is public, so the operation has to say `security: []`", route.Method, route.Path)
			}
		}
	}
}

// The two answers cannot disagree: JSON is the YAML parsed once, and both say the same version.
func TestOpenAPIRoutes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		handler     http.HandlerFunc
		contentType string
	}{
		{"json", httpapi.OpenAPIJSON, "application/json"},
		{"yaml", httpapi.OpenAPIYAML, "application/yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, httptest.NewRequest(http.MethodGet, "/openapi."+tc.name, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tc.contentType) {
				t.Fatalf("content type = %q, want %q", got, tc.contentType)
			}
			if rec.Body.Len() == 0 {
				t.Fatal("empty document")
			}
		})
	}

	rec := httptest.NewRecorder()
	httpapi.OpenAPIJSON(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	var body struct {
		OpenAPI string         `json:"openapi"`
		Paths   map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json does not parse: %v", err)
	}
	if !strings.HasPrefix(body.OpenAPI, "3.1") || len(body.Paths) == 0 {
		t.Fatalf("json document = %+v", body)
	}
}

func scopeList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

// httpapiContract is the embedded document, reachable through the YAML route: the test reads what a
// client reads, not a file path that only the test knows.
func httpapiContract(t *testing.T) []byte {
	t.Helper()
	rec := httptest.NewRecorder()
	httpapi.OpenAPIYAML(rec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	return rec.Body.Bytes()
}
