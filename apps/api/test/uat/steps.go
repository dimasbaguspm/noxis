package uat

import (
	"context"
	"time"

	"github.com/cucumber/godog"

	"noxis/test/uat/kit"
)

// steps registers every scenario step on the world, grouped by business domain.
func (w *world) steps(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		w.reset()
		// A @stub scenario records or replays the shared store instead of calling the API, so it runs with or without the stack.
		if hasTag(s, "@stub") && (envTrue("UAT_STUB") || envTrue("UAT_RECORD")) {
			w.client.Stub = &kit.Stub{Dir: stubDir, Record: envTrue("UAT_RECORD")}
		}
		return ctx, nil
	})
	sc.Step("the stack is running", w.stackRunning)
	w.httpSteps(sc)
	w.snapshotSteps(sc)
	w.memcacheSteps(sc)
	w.mongoSteps(sc)
	w.couchSteps(sc)
	w.searchSteps(sc)
	w.meiliSteps(sc)
	w.connectionSteps(sc)
	w.changeSteps(sc)
	w.driverSteps(sc)
	w.sqlSteps(sc)
	w.erdSteps(sc)
	w.authSteps(sc)
	w.accessSteps(sc)
}

func (w *world) stackRunning(ctx context.Context) error {
	if envTrue("UAT_STUB") {
		return nil // offline replay; there is no stack to wait for
	}
	return waitHTTP(baseURL+"/healthz", 10*time.Second)
}

// hasTag reports whether a scenario carries a tag.
func hasTag(s *godog.Scenario, name string) bool {
	for _, tag := range s.Tags {
		if tag.Name == name {
			return true
		}
	}
	return false
}
