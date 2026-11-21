// Command spec post-processes the document `swag` generated, so the two facts that live in code
// rather than in an annotation land in the document: the scopes a route needs, and nothing else.
//
// The route table is the only authority on scopes — a handler does not check one — so the document
// copies them from `http.Routes()` instead of asking an author to repeat them in a comment that
// nothing compares. `make spec` runs swag and then this; CI regenerates both and fails on drift.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apiserver "noxis/cmd/noxis/http"

	"go.yaml.in/yaml/v3"
)

// docPath is where swag writes, relative to the repository root.
const docPath = "cmd/noxis/http/docs"

func main() {
	if err := run(docPath); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "spec:", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	document, err := read(filepath.Join(dir, "swagger.json"))
	if err != nil {
		return err
	}
	if err := scopes(document); err != nil {
		return err
	}
	if err := payloadSchemas(document); err != nil {
		return err
	}
	return write(dir, document)
}

func read(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // the path is this repository's own docs directory, not input
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return document, nil
}

// scopes copies each route's scopes onto its operation. A route the document does not describe is an
// error here rather than a silent skip: the annotation and the table disagreeing is the defect this
// program exists to prevent.
func scopes(document map[string]any) error {
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return fmt.Errorf("the document has no paths")
	}
	for _, route := range apiserver.Routes(apiserver.Deps{}) {
		item, ok := paths[route.Path].(map[string]any)
		if !ok {
			return fmt.Errorf("%s is served but not documented", route.Path)
		}
		operation, ok := item[strings.ToLower(route.Method)].(map[string]any)
		if !ok {
			return fmt.Errorf("%s %s is served but not documented", route.Method, route.Path)
		}
		if len(route.Scopes) == 0 {
			continue
		}
		declared := make([]any, 0, len(route.Scopes))
		for _, scope := range route.Scopes {
			declared = append(declared, scope)
		}
		operation["x-scopes"] = declared
	}
	return nil
}

func write(dir string, document map[string]any) error {
	body, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "swagger.json"), append(body, '\n'), 0o644); err != nil { //nolint:gosec // a committed document is world-readable on purpose
		return err
	}
	yamlBody, err := yaml.Marshal(document)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "swagger.yaml"), yamlBody, 0o644) //nolint:gosec // a committed document is world-readable on purpose
}

// payloadSchemas wraps each documented 2xx payload in the envelope every answer uses. The annotations
// name the payload — the type the handler actually returns — and the wrapper is applied here, once,
// because `respond.Envelope{data=…}` in twenty comments collides in swag: every override lands in one
// schema named after the field, and the last handler to be parsed wins.
func payloadSchemas(document map[string]any) error {
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return fmt.Errorf("the document has no paths")
	}
	for _, rawPath := range paths {
		item, ok := rawPath.(map[string]any)
		if !ok {
			continue
		}
		for _, rawOperation := range item {
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				continue
			}
			responses, ok := operation["responses"].(map[string]any)
			if !ok {
				continue
			}
			for status, rawResponse := range responses {
				if !strings.HasPrefix(status, "2") {
					continue
				}
				response, ok := rawResponse.(map[string]any)
				if !ok {
					continue
				}
				content, ok := response["content"].(map[string]any)
				if !ok {
					continue
				}
				media, ok := content["application/json"].(map[string]any)
				if !ok {
					continue
				}
				schema, ok := media["schema"].(map[string]any)
				if !ok || !isPayload(schema) {
					continue
				}
				media["schema"] = map[string]any{
					"type": "object",
					"properties": map[string]any{
						"data":  schema,
						"meta":  map[string]any{"$ref": "#/components/schemas/respond.Meta"},
						"error": map[string]any{"$ref": "#/components/schemas/respond.Error"},
					},
				}
			}
		}
	}
	return nil
}

// isPayload reports whether a documented 2xx body belongs under the envelope's `data`. A string body —
// the OpenAPI document answering itself — does not, and an envelope already documented is left alone.
func isPayload(schema map[string]any) bool {
	if ref, ok := schema["$ref"].(string); ok {
		return ref != "#/components/schemas/respond.Envelope"
	}
	kind, _ := schema["type"].(string)
	return kind == "object" || kind == "array"
}
