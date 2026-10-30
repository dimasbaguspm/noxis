package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// driverInfo mirrors one entry of GET /config/drivers.
type driverInfo struct {
	Driver       string   `json:"driver"`
	Label        string   `json:"label"`
	Capabilities []string `json:"capabilities"`
}

// driverSteps prove the catalog is derived from the registered modules rather than from a table someone keeps by hand.
func (w *world) driverSteps(sc *godog.ScenarioContext) {
	sc.Step(`^driver "([^"]*)" advertises exactly "([^"]*)"$`, w.driverAdvertises)
	sc.Step(`^a (GET|POST) to "([^"]*)" for driver "([^"]*)" is refused naming the driver$`, w.capabilityRefused)
	sc.Step("a due change is scheduled on the current connection", w.scheduleDueChange)
	sc.Step("a scheduler tick runs", w.tickScheduler)
	sc.Step("the tick applied both scheduled changes", w.tickAppliedBoth)
}

// driverAdvertises reads the catalog and compares one driver's capability list with what its module implements.
func (w *world) driverAdvertises(ctx context.Context, driver, want string) error {
	var list []driverInfo
	if err := w.api(http.MethodGet, "/external/v1/config/drivers", nil, &list); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("driver catalog: %d %s", w.code, w.body)
	}
	for _, info := range list {
		if info.Driver != driver {
			continue
		}
		if got := strings.Join(info.Capabilities, ","); got != want {
			return fmt.Errorf("driver %q advertises %q, want %q", driver, got, want)
		}
		return nil
	}
	return fmt.Errorf("driver %q is missing from the catalog: %s", driver, w.body)
}

// capabilityRefused asks a driver for a surface its module does not implement: the loader refuses it by driver name instead of reaching the engine.
func (w *world) capabilityRefused(ctx context.Context, method, suffix, driver string) error {
	var body any
	if method == http.MethodPost {
		body = map[string]string{}
	}
	path := enginePathOf(driver, suffix)
	if err := w.api(method, path, body, nil); err != nil {
		return err
	}
	if w.code != http.StatusNotFound {
		return fmt.Errorf("%s %s = %d, want 404: %s", method, path, w.code, w.body)
	}
	if !strings.Contains(w.body, driver) {
		return fmt.Errorf("refusal must name driver %q: %s", driver, w.body)
	}
	return nil
}

// scheduleDueChange files a change whose slot has already arrived and signs it off, so the next tick has one to pick up.
func (w *world) scheduleDueChange(ctx context.Context) error {
	query := "CREATE TABLE IF NOT EXISTS uat_tick (id INT)"
	if err := w.submitScheduledChange(ctx, query, "DROP TABLE uat_tick"); err != nil {
		return err
	}
	if w.change.Status != "validated" && w.change.Status != "unverified" {
		return fmt.Errorf("scheduled change is %q, want validated or unverified", w.change.Status)
	}
	if err := w.approve(ctx); err != nil {
		return err
	}
	w.scheduledIDs = append(w.scheduledIDs, w.changeID)
	return nil
}

// tickAppliedBoth proves each driver's queue claimed only its own change: one tick, two drivers, two applied changes.
func (w *world) tickAppliedBoth(context.Context) error {
	if len(w.scheduledIDs) < 2 {
		return fmt.Errorf("the scenario scheduled %d changes, want 2", len(w.scheduledIDs))
	}
	applied := map[string]bool{}
	for _, id := range w.ticked.Applied {
		applied[id] = true
	}
	for _, id := range w.scheduledIDs {
		if !applied[id] {
			return fmt.Errorf("tick applied %v, missing scheduled change %q", w.ticked.Applied, id)
		}
	}
	return nil
}
