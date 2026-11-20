package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	apiserver "noxis/cmd/noxis/http"
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
	if err := yaml.Unmarshal(openAPIContract(t), &doc); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	return doc
}

// A route the document does not describe is a route nobody can call on purpose; an operation the
// document describes and the server does not serve is worse, because a client writes code against a
// 404. The annotations are the source, so this test is the only thing between them and a lie.
func TestEveryRouteIsDocumented(t *testing.T) {
	doc := load(t)
	if !strings.HasPrefix(doc.OpenAPI, "3.1") {
		t.Fatalf("openapi = %q, want 3.1.x", doc.OpenAPI)
	}
	if doc.Info["title"] == nil || doc.Info["version"] == nil {
		t.Fatal("info needs a title and a version")
	}

	served := make(map[string]bool, len(doc.Paths))
	for _, route := range apiserver.Routes(apiserver.Deps{}) {
		item, ok := doc.Paths[route.Path]
		if !ok {
			t.Fatalf("path %s is served but not documented", route.Path)
		}
		operation, ok := item[strings.ToLower(route.Method)].(map[string]any)
		if !ok {
			t.Fatalf("%s %s is served but not documented", route.Method, route.Path)
		}
		served[strings.ToLower(route.Method)+" "+route.Path] = true

		// A public route must not inherit the document's default security; a route behind a token
		// must name the scheme, or a client cannot know it needs one.
		_, secured := operation["security"]
		if route.Public && secured {
			t.Fatalf("%s %s is public, so the operation cannot require a token", route.Method, route.Path)
		}
		if !route.Public && !secured {
			t.Fatalf("%s %s needs a token, so the operation has to name the scheme", route.Method, route.Path)
		}
	}

	for path, item := range doc.Paths {
		for method := range item {
			if !served[method+" "+path] {
				t.Fatalf("the document describes %s %s, which the route table does not serve", method, path)
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
		{"json", apiserver.OpenAPIJSON, "application/json"},
		{"yaml", apiserver.OpenAPIYAML, "application/yaml"},
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
	apiserver.OpenAPIJSON(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
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

func openAPIContract(t *testing.T) []byte {
	t.Helper()
	rec := httptest.NewRecorder()
	apiserver.OpenAPIYAML(rec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	return rec.Body.Bytes()
}
