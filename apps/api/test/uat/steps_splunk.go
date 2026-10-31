package uat

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"
)

// Splunk scenarios run SPL and review a delete. They are tagged @slow because the image is large and licence-gated, so
// they run only when an operator asks for them.

func (w *world) splunkSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a splunk connection named "([^"]*)"$`, w.registerSplunk)
	sc.Step(`^I search splunk for "([^"]*)"$`, w.splunkSearch)
	sc.Step(`^the splunk result has (\d+) event\(s\)$`, w.splunkEvents)
	sc.Step(`^I request a splunk "([^"]*)" with search "([^"]*)"$`, w.splunkRequestChange)
	sc.Step(`^the splunk request reports an impact of (\d+)$`, w.splunkImpact)
}

// registerSplunk registers the compose splunk service: the management API answers on 8089 with a self-signed certificate.
func (w *world) registerSplunk(ctx context.Context, name string) error {
	conn := map[string]any{
		"name": name, "driver": "splunk", "host": "splunk", "port": 8089,
		"password": "noxis-noxis", "database": "main", "params": map[string]string{"insecure": "true"},
	}
	if err := w.api(http.MethodPost, enginePathOf("splunk", "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register splunk connection: %d", w.code)
	}
	w.conn, w.host, w.targetKey = name, "splunk", "splunk"
	return nil
}

// splunkSearch runs one SPL search through the console's search route.
func (w *world) splunkSearch(ctx context.Context, spl string) error {
	body := map[string]any{"op": "search", "target": "main", "query": spl}
	var out searchResult
	if err := w.api(http.MethodPost, w.enginePath("/connections/"+w.conn+"/search"), body, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("search: %d", w.code)
	}
	w.search = out
	return nil
}

func (w *world) splunkEvents(ctx context.Context, want int) error {
	if len(w.search.Documents) != want {
		return fmt.Errorf("search returned %d event(s), want %d: %v", len(w.search.Documents), want, w.search.Documents)
	}
	return nil
}

// splunkRequestChange submits a delete for review, the way the console does.
func (w *world) splunkRequestChange(ctx context.Context, op, spl string) error {
	w.change = changeBody{}
	body := map[string]any{"connection": w.conn, "notes": "uat " + op, "op": op, "target": "main", "query": spl}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit splunk request: %d", w.code)
	}
	return nil
}

// splunkImpact proves the dry run counted the events the same search matches.
func (w *world) splunkImpact(ctx context.Context, want int) error {
	if w.change.Status != "validated" {
		return fmt.Errorf("request status %q want validated (checks: %+v)", w.change.Status, w.change.Checks)
	}
	if w.change.EstAffected != int64(want) {
		return fmt.Errorf("impact %d want %d (checks: %+v)", w.change.EstAffected, want, w.change.Checks)
	}
	return nil
}
