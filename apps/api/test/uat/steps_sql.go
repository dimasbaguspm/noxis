package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// resultBody is the shared result grid an operation answers with.
type resultBody struct {
	Columns  []string   `json:"columns"`
	Rows     [][]string `json:"rows"`
	Affected int64      `json:"affected"`
	Message  string     `json:"message"`
}

func (w *world) sqlSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I read "([^"]*)"$`, w.read)
	sc.Step(`^I run the write "([^"]*)"$`, w.write)
	sc.Step(`^I send an unknown console operation$`, w.unknownOp)
	sc.Step(`^the request is refused as a bad request$`, w.badRequest)
	sc.Step(`^the result has (\d+) row\(s\)$`, w.rowCount)
	sc.Step(`^the first row's "([^"]*)" is "([^"]*)"$`, w.firstRowCell)
	sc.Step("the request is rejected", w.rejected)
	sc.Step(`^I dry-run "([^"]*)"$`, w.dryRun)
	sc.Step(`^I dry-run "([^"]*)" and it is refused$`, w.dryRunRefused)
	sc.Step(`^analyze on table "([^"]*)" succeeds$`, w.analyze)
	sc.Step(`^the schema lists table "([^"]*)"$`, w.schemaLists)
	sc.Step(`^table "([^"]*)" has (\d+) row\(s\)$`, w.tableRows)
	sc.Step("no scratch schemas remain", w.noScratchSchemas)
}

func (w *world) read(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	return w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/operations", map[string]string{"op": "query", "sql": sqlText}, &w.result)
}

// unknownOp proves the operations route refuses an unknown op instead of guessing.
// badRequest asserts the API rejected the last request as malformed input.
func (w *world) badRequest(context.Context) error {
	if w.code != http.StatusBadRequest {
		return fmt.Errorf("want 400, got %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) unknownOp(ctx context.Context) error {
	w.result = resultBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/operations", map[string]string{"op": "bogus"}, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusBadRequest {
		return fmt.Errorf("unknown op: %d %s", w.code, w.body)
	}
	return nil
}

// write runs a direct DML/DDL statement through the alter route.
func (w *world) write(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/operations", map[string]string{"op": "alter", "sql": sqlText}, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("write: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) rowCount(ctx context.Context, want int) error {
	if w.code != http.StatusOK {
		return fmt.Errorf("read failed: %d", w.code)
	}
	if len(w.result.Rows) != want {
		return fmt.Errorf("rows=%d want %d", len(w.result.Rows), want)
	}
	return nil
}

func (w *world) firstRowCell(ctx context.Context, column, want string) error {
	idx := -1
	for i, c := range w.result.Columns {
		if c == column {
			idx = i
		}
	}
	if idx < 0 {
		return fmt.Errorf("column %q not in %v", column, w.result.Columns)
	}
	if len(w.result.Rows) == 0 {
		return fmt.Errorf("no rows")
	}
	if got := w.result.Rows[0][idx]; got != want {
		return fmt.Errorf("%s=%q want %q", column, got, want)
	}
	return nil
}

func (w *world) rejected(ctx context.Context) error {
	if w.code == http.StatusOK {
		return fmt.Errorf("expected rejection, got 200: %s", w.result.Message)
	}
	return nil
}

func (w *world) dryRun(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/operations", map[string]string{"op": "rollback", "sql": sqlText}, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("dry run: %d", w.code)
	}
	return nil
}

// dryRunRefused expects the driver to refuse a dry run: mysql-family engines commit DDL implicitly, so a rolled-back transaction there would not be a dry run at all.
func (w *world) dryRunRefused(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/operations", map[string]string{"op": "rollback", "sql": sqlText}, &w.result); err != nil {
		return err
	}
	if w.code == http.StatusOK {
		return fmt.Errorf("dry run of DDL must be refused on this driver: %s", w.result.Message)
	}
	return nil
}

func (w *world) analyze(ctx context.Context, table string) error {
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/operations", map[string]string{"op": "analyze", "table": table}, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("analyze: %d", w.code)
	}
	return nil
}

func (w *world) schemaLists(ctx context.Context, table string) error {
	var body struct {
		Tables []struct {
			Name string `json:"name"`
		} `json:"tables"`
	}
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/connections/"+w.conn+"/schema", nil, &body); err != nil {
		return err
	}
	for _, t := range body.Tables {
		if t.Name == table {
			return nil
		}
	}
	return fmt.Errorf("table %q not in schema (%d tables)", table, len(body.Tables))
}

// noScratchSchemas proves cleanup: no verification schema survived, pass or fail.
func (w *world) noScratchSchemas(ctx context.Context) error {
	var body struct {
		Tables []struct {
			Schema string `json:"schema"`
		} `json:"tables"`
	}
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/connections/"+w.conn+"/schema", nil, &body); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("schema list: %d", w.code)
	}
	for _, t := range body.Tables {
		if strings.HasPrefix(t.Schema, "noxis_verify_") {
			return fmt.Errorf("orphan scratch schema %q", t.Schema)
		}
	}
	return nil
}

func (w *world) tableRows(ctx context.Context, table string, want int) error {
	w.result = resultBody{}
	// no schema parameter: the driver's own default resolves the table, which is what keeps the step driver-agnostic
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/connections/"+w.conn+"/tables/"+table+"/rows", nil, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("rows: %d", w.code)
	}
	if len(w.result.Rows) != want {
		return fmt.Errorf("table %s rows=%d want %d", table, len(w.result.Rows), want)
	}
	return nil
}
