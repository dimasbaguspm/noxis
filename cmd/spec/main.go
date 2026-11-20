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
