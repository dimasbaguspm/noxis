package uat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"

	"noxis/test/uat/kit"
)

// genericSteps is the vocabulary every endpoint file leans on: reset the registry, take a session or drop it, send a
// request with a body and headers, and assert on the status or the body. A domain step still exists where a domain has
// to be set up, but the request and the assertion are the same everywhere.
func (w *world) genericSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a clean registry$`, w.cleanRegistry)
	sc.Step(`^a fresh session$`, w.freshSession)
	sc.Step(`^no session$`, w.noSession)
	sc.Step(`^I send a (GET|POST|PUT|PATCH|DELETE) to "([^"]*)"$`, w.sendSimple)
	sc.Step(`^I send a (GET|POST|PUT|PATCH|DELETE) to "([^"]*)" with body:$`, w.sendBody)
	sc.Step(`^I send a (GET|POST|PUT|PATCH|DELETE) to "([^"]*)" with headers:$`, w.sendWithHeaders)
	sc.Step(`^the response body contains "([^"]*)"$`, w.bodyContains)
	sc.Step(`^the response body does not contain "([^"]*)"$`, w.bodyMisses)
	sc.Step(`^the response field "([^"]*)" is "([^"]*)"$`, w.responseFieldIs)
}

// cleanRegistry deletes every registered connection, which is the only state the suite mutates across scenarios: a
// change or a grant hangs off a connection, so removing the connections removes them too.
func (w *world) cleanRegistry(ctx context.Context) error {
	return pruneConnections()
}

// freshSession signs in again and installs the token: an account holds one session, so a scenario that wants a known
// starting session asks for one instead of inheriting whatever the previous scenario left.
func (w *world) freshSession(ctx context.Context) error {
	_, err := w.signIn(adminEmail, adminPassword)
	return err
}

// noSession drops the bearer for the rest of the scenario.
func (w *world) noSession(ctx context.Context) error {
	w.headers["Authorization"] = ""
	return nil
}

func (w *world) sendSimple(ctx context.Context, method, path string) error {
	return w.send(kit.Request{Method: method, Path: path})
}

// sendBody sends a request whose body is the step's docstring, verbatim.
func (w *world) sendBody(ctx context.Context, method, path string, payload *godog.DocString) error {
	return w.send(kit.Request{Method: method, Path: path, Body: []byte(payload.Content)})
}

// sendWithHeaders sends a request after folding the step's table into the per-scenario headers.
func (w *world) sendWithHeaders(ctx context.Context, method, path string, table *godog.Table) error {
	for _, row := range table.Rows {
		if len(row.Cells) < 2 {
			return fmt.Errorf("header row %v needs a name and a value", row.Cells)
		}
		w.headers[row.Cells[0].Value] = row.Cells[1].Value
	}
	return w.send(kit.Request{Method: method, Path: path})
}

func (w *world) bodyContains(ctx context.Context, want string) error {
	if !strings.Contains(w.body, want) {
		return fmt.Errorf("response body does not contain %q: %s", want, w.body)
	}
	return nil
}

func (w *world) bodyMisses(ctx context.Context, unwanted string) error {
	if strings.Contains(w.body, unwanted) {
		return fmt.Errorf("response body contains %q: %s", unwanted, w.body)
	}
	return nil
}

// responseFieldIs reads one field of the last response by a dotted path, so an endpoint file asserts a value instead of
// a substring wherever the value is structured.
func (w *world) responseFieldIs(ctx context.Context, path, want string) error {
	var doc any
	if err := w.resp.Decode(&doc); err != nil {
		return fmt.Errorf("decode %s: %w (%s)", path, err, w.body)
	}
	got, err := fieldAt(doc, path)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("field %s = %q, want %q (%s)", path, got, want, w.body)
	}
	return nil
}

// fieldAt walks a dotted path through a decoded JSON document; a numeric segment indexes a list.
func fieldAt(doc any, path string) (string, error) {
	current := doc
	for _, segment := range strings.Split(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[segment]
			if !ok {
				return "", fmt.Errorf("field %q is absent", path)
			}
			current = value
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) {
				return "", fmt.Errorf("field %q is not an index into a list of %d", path, len(node))
			}
			current = node[index]
		default:
			return "", fmt.Errorf("field %q runs past a %T", path, current)
		}
	}
	switch value := current.(type) {
	case string:
		return value, nil
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(value), nil
	case nil:
		return "", nil
	}
	return "", fmt.Errorf("field %q is a %T, not a scalar", path, current)
}
