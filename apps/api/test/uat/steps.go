package uat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cucumber/godog"

	"noxis/test/uat/kit"
)

// changeBody is the wire shape of a change (only the fields UAT asserts on).
type changeBody struct {
	ID            string      `json:"id"`
	Notes         string      `json:"notes"`
	Status        string      `json:"status"`
	Checks        []checkBody `json:"checks"`
	EstDurationMs int64       `json:"estDurationMs"`
	EstAffected   int64       `json:"estAffected"`
	ScratchSchema string      `json:"scratchSchema"`
	BatchID       string      `json:"batchId"`
}

// steps registers every scenario step on the world.
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

	sc.Step(`^connection "([^"]*)" is registered$`, w.registerConnection)
	sc.Step(`^connection "([^"]*)" is registered for driver "([^"]*)"$`, w.registerConnectionDriver)
	sc.Step("the connection test passes", w.testConnection)
	sc.Step("the connection list hides the password", w.listRedacted)
	sc.Step(`^the operator console is served at "/"$`, w.consoleServed)

	sc.Step(`^I read "([^"]*)"$`, w.read)
	sc.Step(`^the result has (\d+) row\(s\)$`, w.rowCount)
	sc.Step(`^the first row's "([^"]*)" is "([^"]*)"$`, w.firstRowCell)
	sc.Step("the request is rejected", w.rejected)

	sc.Step(`^I submit a change with query "([^"]*)" and rollback "([^"]*)"$`, w.submitChange)
	sc.Step(`^the change is "([^"]*)"$`, w.changeStatus)
	sc.Step("I approve the change", w.approve)
	sc.Step("I apply the change", w.apply)
	sc.Step(`^I submit a change scheduled at now with query "([^"]*)" and rollback "([^"]*)"$`, w.submitScheduledChange)
	sc.Step("a scheduler tick applies the due change", w.tickScheduler)
	sc.Step("applying is rejected", w.applyRejected)
	sc.Step("I reject the change", w.rejectChange)
	sc.Step("the change list still contains it", w.changeStillPresent)
	sc.Step("the change was not recorded", w.changeNotRecorded)
	sc.Step("the change was verified in a scratch schema", w.changeScratchSchema)
	sc.Step("no scratch schemas remain", w.noScratchSchemas)
	sc.Step(`^the change records an estimated duration and affected count$`, w.changeEstimates)

	sc.Step(`^I dry-run "([^"]*)"$`, w.dryRun)
	sc.Step(`^I dry-run "([^"]*)" and it is refused$`, w.dryRunRefused)
	sc.Step(`^analyze on table "([^"]*)" succeeds$`, w.analyze)
	sc.Step(`^the schema lists table "([^"]*)"$`, w.schemaLists)
	sc.Step(`^table "([^"]*)" has (\d+) row\(s\)$`, w.tableRows)
	sc.Step(`^the ERD of "([^"]*)" marks "([^"]*)" as a primary key$`, w.erdPrimaryKey)
	sc.Step(`^the ERD of "([^"]*)" links "([^"]*)" to "([^"]*)"$`, w.erdLinks)
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

// registerConnection upserts the default SQL target as a noxis connection.
func (w *world) registerConnection(ctx context.Context, name string) error {
	return w.registerSQL(ctx, name, "postgres")
}

// registerConnectionDriver registers a non-postgres SQL target.
func (w *world) registerConnectionDriver(ctx context.Context, name, driver string) error {
	return w.registerSQL(ctx, name, driver)
}

// registerSQL builds the connection from the target the compose file declares.
func (w *world) registerSQL(ctx context.Context, name, key string) error {
	t := targets[key]
	conn := map[string]any{
		"name": name, "driver": t.Driver, "host": t.Host, "port": t.Port,
		"user": t.User, "password": t.Password, "database": t.Database,
	}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register %s connection: %d", key, w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, key
	return nil
}

func (w *world) testConnection(ctx context.Context) error {
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/test", map[string]string{}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("test connection: %d", w.code)
	}
	return nil
}

func (w *world) listRedacted(ctx context.Context) error {
	code, body, err := w.raw("/external/v1/engine/sql/connections")
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("list connections: %d", code)
	}
	// the wire omits the field entirely (json:"password,omitempty"), so the strongest check is that no key named password is present at all — not a substring match, which a host name or a subject can trip.
	if strings.Contains(strings.ToLower(body), `"password"`) {
		return fmt.Errorf("connection list must not carry a password: %s", body)
	}
	if !strings.Contains(body, `"name"`) {
		return fmt.Errorf("connection list looks wrong: %s", body)
	}
	return nil
}

// consoleServed asserts the root path returns the embedded console shell.
func (w *world) consoleServed(ctx context.Context) error {
	code, body, err := w.raw("/")
	if err != nil {
		return err
	}
	// case-insensitive: the brand is "noXis", the binary is "noxis"
	if code != http.StatusOK || !strings.Contains(strings.ToLower(body), "noxis") {
		return fmt.Errorf("console at /: status %d, body %q", code, body[:min(len(body), 60)])
	}
	return nil
}

func (w *world) read(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	return w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/query", map[string]string{"sql": sqlText}, &w.result)
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
		return errors.New("no rows")
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

func (w *world) submitChange(ctx context.Context, query, rollback string) error {
	w.change = changeBody{}
	body := map[string]string{
		"connection": w.conn, "notes": "uat change", "query": query, "rollback": rollback,
	}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/changes", body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit change: %d", w.code)
	}
	w.changeID = w.change.ID
	return nil
}

func (w *world) changeStatus(ctx context.Context, want string) error {
	if w.change.Status != want {
		return fmt.Errorf("change status %q want %q (checks: %v)", w.change.Status, want, w.change.Checks)
	}
	return nil
}

func (w *world) approve(ctx context.Context) error {
	return w.changeAction("/approve")
}

func (w *world) apply(ctx context.Context) error {
	return w.changeAction("/apply")
}

// submitScheduledChange is submitChange with a due schedule, so the tick has something to pick up without waiting.
func (w *world) submitScheduledChange(ctx context.Context, query, rollback string) error {
	w.change = changeBody{}
	body := map[string]any{
		"connection": w.conn, "notes": "uat change", "query": query, "rollback": rollback,
		"scheduledAt": time.Now().Unix(),
	}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/changes", body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit scheduled change: %d", w.code)
	}
	w.changeID = w.change.ID
	return nil
}

func (w *world) tickScheduler(ctx context.Context) error {
	if err := w.tick(ctx); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("scheduler tick: status %d body %s", w.code, w.body)
	}
	return nil
}

// tick drives POST /internal/callback/scheduler and, on success, reloads the change under test so the next status assertion sees the result.
func (w *world) tick(ctx context.Context) error {
	if err := w.send(kit.Request{
		Method: http.MethodPost,
		Path:   "/internal/callback/scheduler",
		Sign:   signScheduler,
	}); err != nil {
		return err
	}
	w.ticked = tickBody{}
	_ = w.resp.Decode(&w.ticked)
	if w.code != http.StatusOK || w.changeID == "" {
		return nil
	}
	w.change = changeBody{}
	return w.api(http.MethodGet, "/external/v1/engine/sql/changes/"+w.changeID, nil, &w.change)
}

// changeAction POSTs to a change sub-resource and records the new state.
func (w *world) changeAction(action string) error {
	w.change = changeBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/changes/"+w.changeID+action, map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("%s: %d", action, w.code)
	}
	return nil
}

func (w *world) applyRejected(ctx context.Context) error {
	w.change = changeBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/changes/"+w.changeID+"/apply", map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code == http.StatusOK {
		return errors.New("apply must require approval first")
	}
	return nil
}

func (w *world) rejectChange(ctx context.Context) error {
	return w.changeAction("/reject")
}

// changeStillPresent proves rejection keeps the artifact (no hard delete).
func (w *world) changeStillPresent(ctx context.Context) error {
	var body struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/changes", nil, &body); err != nil {
		return err
	}
	for _, c := range body.Changes {
		if c.ID == w.changeID {
			return nil
		}
	}
	return fmt.Errorf("change %s disappeared from the list", w.changeID)
}

// changeNotRecorded proves rejected proposals never enter the store: the submit response carries no id and the list does not grow.
func (w *world) changeNotRecorded(ctx context.Context) error {
	if w.changeID != "" {
		return fmt.Errorf("failed change kept an id: %s", w.changeID)
	}
	var body struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/changes", nil, &body); err != nil {
		return err
	}
	for _, c := range body.Changes {
		if c.Notes == "uat change" && c.Status == "failed" {
			return fmt.Errorf("failed change was recorded: %+v", c)
		}
	}
	return nil
}

func (w *world) changeScratchSchema(ctx context.Context) error {
	if !strings.HasPrefix(w.change.ScratchSchema, "noxis_verify_") {
		return fmt.Errorf("change was not verified in a scratch schema: %q", w.change.ScratchSchema)
	}
	return nil
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

func (w *world) changeEstimates(ctx context.Context) error {
	if w.change.EstDurationMs < 0 || w.change.EstAffected < 0 {
		return fmt.Errorf("negative estimates: %+v", w.change)
	}
	if w.change.Status != "applied" {
		return fmt.Errorf("estimates are recorded on validation; status=%s", w.change.Status)
	}
	return nil
}

func (w *world) dryRun(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/rollback", map[string]string{"sql": sqlText}, &w.result); err != nil {
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
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/rollback", map[string]string{"sql": sqlText}, &w.result); err != nil {
		return err
	}
	if w.code == http.StatusOK {
		return fmt.Errorf("dry run of DDL must be refused on this driver: %s", w.result.Message)
	}
	return nil
}

func (w *world) analyze(ctx context.Context, table string) error {
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections/"+w.conn+"/analyze", map[string]string{"table": table}, &w.result); err != nil {
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

// erdBody is the wire shape of the ERD endpoint (only the asserted fields).
type erdBody struct {
	Connection struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	} `json:"connection"`
	Tables []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name       string `json:"name"`
			PrimaryKey bool   `json:"primaryKey"`
		} `json:"columns"`
	} `json:"tables"`
	Mermaid string `json:"mermaid"`
}

// erd loads one connection's schema graph.
func (w *world) erd(ctx context.Context, connection string) (erdBody, string, error) {
	var body erdBody
	code, raw, err := w.raw("/external/v1/engine/sql/connections/" + connection + "/erd")
	if err != nil {
		return body, "", err
	}
	if code != http.StatusOK {
		return body, raw, fmt.Errorf("erd: %d %s", code, raw)
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return body, raw, fmt.Errorf("erd decode: %w", err)
	}
	if body.Connection.Name != connection {
		return body, raw, fmt.Errorf("erd names connection %q, want %q", body.Connection.Name, connection)
	}
	if strings.Contains(strings.ToLower(raw), `"password"`) {
		return body, raw, fmt.Errorf("the erd must not carry a password: %s", raw)
	}
	return body, raw, nil
}

// erdPrimaryKey asserts the catalog's primary key reaches the diagram.
func (w *world) erdPrimaryKey(ctx context.Context, connection, column string) error {
	body, _, err := w.erd(ctx, connection)
	if err != nil {
		return err
	}
	for _, t := range body.Tables {
		for _, c := range t.Columns {
			if c.Name == column && c.PrimaryKey {
				return nil
			}
		}
	}
	return fmt.Errorf("no column %q is marked as a primary key in %+v", column, body.Tables)
}

// erdLinks asserts a declared foreign key becomes a relationship in the Mermaid output, pointing from the parent to the child.
func (w *world) erdLinks(ctx context.Context, connection, child, parent string) error {
	body, _, err := w.erd(ctx, connection)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(body.Mermaid, "\n") {
		if strings.Contains(line, "||--") && strings.Contains(line, parent) && strings.Contains(line, child) {
			return nil
		}
	}
	return fmt.Errorf("no relationship from %q to %q in:\n%s", parent, child, body.Mermaid)
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
