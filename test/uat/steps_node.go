package uat

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cucumber/godog"
)

// stepsNode registers the node lifecycle steps (stack, registration,
// assignment, convergence, status).
func (w *world) stepsNode(sc *godog.ScenarioContext) {
	sc.Step("the stack is running", w.stackRunning)
	sc.Step(`node "{}" is registered`, w.nodeRegistered)
	sc.Step(`the master assigns engine "{}" version "{}" to node "{}"`, w.assignEngine)
	sc.Step(`the master tries to assign engine "{}" version "{}" to node "{}"`, w.tryAssign)
	sc.Step("the master rejects the assignment", w.assignRejected)
	sc.Step(`the instance "{}" container is running image "{}"`, w.containerImage)
	sc.Step(`the node "{}" status is "{}"`, w.nodeStatus)
	sc.Step(`the node "{}" status detail mentions "{}"`, w.statusMentions)
}

// stackRunning asserts the compose stack is up (stackUp ran in the suite).
func (w *world) stackRunning(ctx context.Context) error {
	return waitHTTP(masterURL+"/healthz", http.StatusOK, 10*time.Second)
}

// nodeRegistered waits until the master lists the node healthy. The compose
// slave registers itself on boot (its entrypoint loops until it succeeds), so
// this scenario waits, it does not register.
func (w *world) nodeRegistered(ctx context.Context, name string) error {
	return w.waitState(name, "up", waitRegister)
}

// assignEngine publishes a new desired spec (the slave converges async).
func (w *world) assignEngine(ctx context.Context, engine, version, node string) error {
	spec := map[string]any{"engines": []map[string]any{
		{"name": engine, "type": engine, "version": version},
	}}
	code, b, err := w.api(http.MethodPut, "/v1/nodes/"+node+"/spec", spec)
	if err != nil {
		return err
	}
	if code != http.StatusNoContent {
		return fmt.Errorf("assign %s: %d %s", node, code, b)
	}
	return nil
}

// tryAssign attempts the same assign but records the status for an assertion
// step (used for rejection paths) instead of failing immediately.
func (w *world) tryAssign(ctx context.Context, engine, version, node string) error {
	spec := map[string]any{"engines": []map[string]any{
		{"name": engine, "type": engine, "version": version},
	}}
	code, _, err := w.api(http.MethodPut, "/v1/nodes/"+node+"/spec", spec)
	if err != nil {
		return err
	}
	w.code = code
	return nil
}

func (w *world) assignRejected(ctx context.Context) error {
	if w.code != http.StatusBadRequest {
		return fmt.Errorf("expected 400 for unregistered node, got %d", w.code)
	}
	return nil
}

func (w *world) containerImage(ctx context.Context, name, image string) error {
	return w.waitContainer(name, image, waitConverge)
}

func (w *world) nodeStatus(ctx context.Context, name, state string) error {
	return w.waitState(name, state, waitRegister)
}

func (w *world) statusMentions(ctx context.Context, name, needle string) error {
	return w.waitDetail(name, needle, waitRegister)
}
