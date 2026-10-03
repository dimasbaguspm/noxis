package uat

import (
	"context"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
)

// stepsDoorman registers the doorman SQL op steps (query/alter/rollback/
// analytics) and their assertions.
func (w *world) stepsDoorman(sc *godog.ScenarioContext) {
	sc.Step(`I {} engine "{}" with "{}"`, w.opEngine)
	sc.Step(`I analyze engine "{}" on table "{}"`, w.analyzeTable)
	sc.Step(`the output is "{}"`, w.outputIs)
	sc.Step("the master rejects the request with a 400", w.rejected400)
	sc.Step("the dry-run reports success", w.dryRunOK)
	sc.Step("the dry-run reports an error", w.dryRunErr)
	sc.Step("the analyze succeeds", w.analyzeOK)
	sc.Step(`column "{}" still exists in table "{}" on engine "{}"`, w.columnExists)
}

// opEngine maps the step's verb to the doorman op name and remembers the
// verdict for the following assertion step.
func (w *world) opEngine(ctx context.Context, op, engine, sql string) error {
	return w.doorman(op, engine, sql)
}

func (w *world) analyzeTable(ctx context.Context, engine, table string) error {
	path := "/v1/nodes/" + nodeName + "/doorman/analytics"
	code, out, err := w.pollOp(path, map[string]string{"engine": engine, "table": table})
	if err != nil {
		return err
	}
	w.code, w.out = code, out
	return nil
}

// outputIs asserts the last op produced exactly `want` (postgres -t -A).
func (w *world) outputIs(ctx context.Context, want string) error {
	if w.out != want {
		return fmt.Errorf("output %q, want %q", w.out, want)
	}
	return nil
}

func (w *world) rejected400(ctx context.Context) error {
	if w.code != 400 {
		return fmt.Errorf("expected 400 (guard rejection), got %d %q", w.code, w.out)
	}
	return nil
}

// dryRunOK: the tx wrapper prints BEGIN…ROLLBACK command tags, so a
// successful dry run mentions ROLLBACK.
func (w *world) dryRunOK(ctx context.Context) error {
	if w.code != 200 || !strings.Contains(w.out, "ROLLBACK") {
		return fmt.Errorf("dry run: status %d %q", w.code, w.out)
	}
	return nil
}

func (w *world) dryRunErr(ctx context.Context) error {
	if w.code == 200 {
		return fmt.Errorf("dry run should have failed, got 200 %q", w.out)
	}
	return nil
}

func (w *world) analyzeOK(ctx context.Context) error {
	if w.code != 200 {
		return fmt.Errorf("analyze: status %d %q", w.code, w.out)
	}
	return nil
}

// columnExists probes the catalog (information_schema is postgres; the
// features are postgres-only).
func (w *world) columnExists(ctx context.Context, column, table, engine string) error {
	q := fmt.Sprintf("SELECT count(*) FROM information_schema.columns WHERE table_name='%s' AND column_name='%s'", table, column)
	if err := w.doorman("query", engine, q); err != nil {
		return err
	}
	return w.outputIs(ctx, "1")
}
