package http_test

import (
	"strings"
	"testing"
)

// wants is the type each route answers under the envelope's `data`. The handler annotation names it and
// cmd/spec wraps it into the envelope; this table is the reader that fails when the two drift apart, and
// it fails both ways — an operation with no expectation here, and an expectation with no operation.
var wants = map[string]string{
	"GET /healthz":             "handlers.Health",
	"GET /openapi.json":        "string",
	"GET /openapi.yaml":        "string",
	"GET /v1/whoami":           "handlers.Whoami",
	"GET /v1/version":          "handlers.VersionInfo",
	"GET /v1/targets":          "[]handlers.Target",
	"POST /v1/targets":         "handlers.Target",
	"POST /v1/targets/test":    "handlers.TestResult",
	"GET /v1/targets/{ref}":    "handlers.Target",
	"PATCH /v1/targets/{ref}":  "handlers.Target",
	"DELETE /v1/targets/{ref}": "handlers.Target",

	"POST /v1/targets/{ref}/probe":                        "handlers.ProbeResult",
	"GET /v1/targets/{ref}/capabilities":                  "domain.Capabilities",
	"POST /v1/targets/{ref}/query":                        "domain.Rows",
	"GET /v1/targets/{ref}/databases":                     "[]string",
	"GET /v1/targets/{ref}/databases/{db}/tables":         "[]domain.Table",
	"GET /v1/targets/{ref}/databases/{db}/tables/{table}": "domain.TableDetail",

	"POST /v1/targets/{ref}/proposals":  "handlers.Proposal",
	"GET /v1/proposals":                 "[]handlers.Proposal",
	"GET /v1/proposals/{ref}":           "handlers.ProposalDetail",
	"POST /v1/proposals/{ref}/rehearse": "handlers.ProposalDetail",
	"POST /v1/proposals/{ref}/approve":  "handlers.Proposal",
	"POST /v1/proposals/{ref}/reject":   "handlers.Proposal",
	"POST /v1/proposals/{ref}/apply":    "handlers.Proposal",
	"GET /v1/proposals/{ref}/events":    "[]handlers.JournalEntry",
}

// The document names what each operation answers, not `object`: a client generator that cannot see the
// payload writes code against nothing, and a schema nobody returns is worse than no schema at all.
func TestEveryPayloadIsDocumented(t *testing.T) {
	doc := load(t)
	seen := make(map[string]bool, len(wants))

	for key, want := range wants {
		method, path, ok := strings.Cut(key, " ")
		if !ok {
			t.Fatalf("%q is not \"METHOD path\"", key)
		}
		item, ok := doc.Paths[path]
		if !ok {
			t.Fatalf("%s is documented nowhere", key)
		}
		operation, ok := item[strings.ToLower(method)].(map[string]any)
		if !ok {
			t.Fatalf("%s is documented nowhere", key)
		}
		seen[key] = true

		schema, ok := successSchema(operation)
		if !ok {
			t.Fatalf("%s has no documented 2xx body", key)
		}
		if got := dataType(schema); got != want {
			t.Fatalf("%s answers %q, the document says %q", key, want, got)
		}
	}

	for path, item := range doc.Paths {
		for method, rawOperation := range item {
			if _, ok := rawOperation.(map[string]any); !ok {
				continue
			}
			if key := strings.ToUpper(method) + " " + path; !seen[key] {
				t.Fatalf("%s is documented but this test has no expectation for its payload", key)
			}
		}
	}
}

// successSchema is the schema of the first 2xx response an operation documents.
func successSchema(operation map[string]any) (map[string]any, bool) {
	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		return nil, false
	}
	for _, status := range []string{"200", "201", "202", "204"} {
		response, ok := responses[status].(map[string]any)
		if !ok {
			continue
		}
		content, ok := response["content"].(map[string]any)
		if !ok {
			return nil, false
		}
		return mediaSchema(content)
	}
	return nil, false
}

// mediaSchema is the documented body whatever it is produced as: the YAML route answers
// application/yaml, and a body documented under another media type is still a documented body.
func mediaSchema(content map[string]any) (map[string]any, bool) {
	if media, ok := content["application/json"].(map[string]any); ok {
		schema, ok := media["schema"].(map[string]any)
		return schema, ok
	}
	for _, raw := range content {
		media, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if schema, ok := media["schema"].(map[string]any); ok {
			return schema, true
		}
	}
	return nil, false
}

// dataType reads the payload's type name out of a documented response: the envelope's `data` when the
// answer is wrapped, the schema itself when it is not.
func dataType(schema map[string]any) string {
	if properties, ok := schema["properties"].(map[string]any); ok {
		if data, ok := properties["data"].(map[string]any); ok {
			return typeName(data)
		}
	}
	return typeName(schema)
}

func typeName(schema map[string]any) string {
	if ref, ok := schema["$ref"].(string); ok {
		return strings.TrimPrefix(ref, "#/components/schemas/")
	}
	kind, _ := schema["type"].(string)
	if kind != "array" {
		return kind
	}
	if items, ok := schema["items"].(map[string]any); ok {
		return "[]" + typeName(items)
	}
	return kind
}
