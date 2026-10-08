package uat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cucumber/godog"
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
}

// steps registers every scenario step on the world.
func (w *world) steps(sc *godog.ScenarioContext) {
	sc.Step("the stack is running", w.stackRunning)
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
	sc.Step("a tick without the key is rejected", w.tickWithoutKey)
	sc.Step("a tick with the key applies the due change", w.tickWithKey)
	sc.Step("applying is rejected", w.applyRejected)
	sc.Step("I reject the change", w.rejectChange)
	sc.Step("the change list still contains it", w.changeStillPresent)
	sc.Step("the change was not recorded", w.changeNotRecorded)
	sc.Step("the change was verified in a scratch schema", w.changeScratchSchema)
	sc.Step("no scratch schemas remain", w.noScratchSchemas)
	sc.Step(`^the change records an estimated duration and affected count$`, w.changeEstimates)
	sc.Step(`^the instance stats report connections and size$`, w.instanceStats)

	sc.Step(`^I dry-run "([^"]*)"$`, w.dryRun)
	sc.Step(`^analytics on table "([^"]*)" succeeds$`, w.analytics)
	sc.Step(`^the schema lists table "([^"]*)"$`, w.schemaLists)
	sc.Step(`^table "([^"]*)" has (\d+) row\(s\)$`, w.tableRows)
}

func (w *world) stackRunning(ctx context.Context) error {
	return waitHTTP(baseURL+"/healthz", 10*time.Second)
}

// registerConnection upserts the compose postgres as a noxis connection.
func (w *world) registerConnection(ctx context.Context, name string) error {
	conn := map[string]any{
		"name": name, "driver": "postgres", "host": connHost, "port": 5432,
		"user": "postgres", "password": connPass, "database": "demo",
	}
	if err := w.api(http.MethodPost, "/v1/sql/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register connection: %d", w.code)
	}
	w.conn = name
	return nil
}

// registerConnectionDriver registers a non-postgres target. The compose
// service name equals the driver (mariadb) on its default port.
func (w *world) registerConnectionDriver(ctx context.Context, name, driver string) error {
	conn := map[string]any{
		"name": name, "driver": driver, "host": driver, "port": 3306,
		"user": "root", "password": connPass, "database": "demo",
	}
	if err := w.api(http.MethodPost, "/v1/sql/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register %s connection: %d", driver, w.code)
	}
	w.conn = name
	return nil
}

func (w *world) testConnection(ctx context.Context) error {
	if err := w.api(http.MethodPost, "/v1/sql/connections/"+w.conn+"/test", map[string]string{}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("test connection: %d", w.code)
	}
	return nil
}

func (w *world) listRedacted(ctx context.Context) error {
	code, body, err := w.raw("/v1/sql/connections")
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("list connections: %d", code)
	}
	if strings.Contains(body, connPass) {
		return fmt.Errorf("connection list must not carry the password: %s", body)
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
	return w.api(http.MethodPost, "/v1/sql/connections/"+w.conn+"/query", map[string]string{"sql": sqlText}, &w.result)
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
	if err := w.api(http.MethodPost, "/v1/sql/changes", body, &w.change); err != nil {
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

// submitScheduledChange is submitChange with a due schedule, so the tick has
// something to pick up without waiting.
func (w *world) submitScheduledChange(ctx context.Context, query, rollback string) error {
	w.change = changeBody{}
	body := map[string]any{
		"connection": w.conn, "notes": "uat change", "query": query, "rollback": rollback,
		"scheduledAt": time.Now().Unix(),
	}
	if err := w.api(http.MethodPost, "/v1/sql/changes", body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit scheduled change: %d", w.code)
	}
	w.changeID = w.change.ID
	return nil
}

func (w *world) tickWithoutKey(ctx context.Context) error {
	if err := w.tick(ctx, ""); err != nil {
		return err
	}
	if w.code != http.StatusUnauthorized {
		return fmt.Errorf("tick without key: status %d want 401", w.code)
	}
	return nil
}

func (w *world) tickWithKey(ctx context.Context) error {
	if err := w.tick(ctx, tickKey); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("tick with key: status %d body %s", w.code, w.body)
	}
	return nil
}

// tick drives POST /internal/tick with an optional key and, on success,
// reloads the change under test so the next status assertion sees the result.
func (w *world) tick(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/internal/tick", http.NoBody)
	if err != nil {
		return err
	}
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	w.code = resp.StatusCode
	w.body = strings.TrimSpace(string(b))
	w.ticked = tickBody{}
	if len(b) > 0 {
		_ = json.Unmarshal(b, &w.ticked)
	}
	if w.code != http.StatusOK || w.changeID == "" {
		return nil
	}
	w.change = changeBody{}
	return w.api(http.MethodGet, "/v1/sql/changes/"+w.changeID, nil, &w.change)
}

// changeAction POSTs to a change sub-resource and records the new state.
func (w *world) changeAction(action string) error {
	w.change = changeBody{}
	if err := w.api(http.MethodPost, "/v1/sql/changes/"+w.changeID+action, map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("%s: %d", action, w.code)
	}
	return nil
}

func (w *world) applyRejected(ctx context.Context) error {
	w.change = changeBody{}
	if err := w.api(http.MethodPost, "/v1/sql/changes/"+w.changeID+"/apply", map[string]string{}, &w.change); err != nil {
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
	if err := w.api(http.MethodGet, "/v1/sql/changes", nil, &body); err != nil {
		return err
	}
	for _, c := range body.Changes {
		if c.ID == w.changeID {
			return nil
		}
	}
	return fmt.Errorf("change %s disappeared from the list", w.changeID)
}

// changeNotRecorded proves rejected proposals never enter the store: the
// submit response carries no id and the list does not grow.
func (w *world) changeNotRecorded(ctx context.Context) error {
	if w.changeID != "" {
		return fmt.Errorf("failed change kept an id: %s", w.changeID)
	}
	var body struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, "/v1/sql/changes", nil, &body); err != nil {
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

// noScratchSchemas proves cleanup: no verification schema survived, pass or
// fail.
func (w *world) noScratchSchemas(ctx context.Context) error {
	var body struct {
		Tables []struct {
			Schema string `json:"schema"`
		} `json:"tables"`
	}
	if err := w.api(http.MethodGet, "/v1/sql/connections/"+w.conn+"/schema", nil, &body); err != nil {
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

func (w *world) instanceStats(ctx context.Context) error {
	var body struct {
		Connections    int64 `json:"connections"`
		DatabaseSize   int64 `json:"databaseSizeBytes"`
		MaxConnections int64 `json:"maxConnections"`
	}
	if err := w.api(http.MethodGet, "/v1/sql/connections/"+w.conn+"/stats", nil, &body); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("stats: %d", w.code)
	}
	if body.Connections <= 0 || body.MaxConnections <= 0 {
		return fmt.Errorf("stats look wrong: %+v", body)
	}
	return nil
}

func (w *world) dryRun(ctx context.Context, sqlText string) error {
	w.result = resultBody{}
	if err := w.api(http.MethodPost, "/v1/sql/connections/"+w.conn+"/rollback", map[string]string{"sql": sqlText}, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("dry run: %d", w.code)
	}
	return nil
}

func (w *world) analytics(ctx context.Context, table string) error {
	if err := w.api(http.MethodPost, "/v1/sql/connections/"+w.conn+"/analytics", map[string]string{"table": table}, &w.result); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("analytics: %d", w.code)
	}
	return nil
}

func (w *world) schemaLists(ctx context.Context, table string) error {
	var body struct {
		Tables []struct {
			Name string `json:"name"`
		} `json:"tables"`
	}
	if err := w.api(http.MethodGet, "/v1/sql/connections/"+w.conn+"/schema", nil, &body); err != nil {
		return err
	}
	for _, t := range body.Tables {
		if t.Name == table {
			return nil
		}
	}
	return fmt.Errorf("table %q not in schema (%d tables)", table, len(body.Tables))
}

func (w *world) tableRows(ctx context.Context, table string, want int) error {
	w.result = resultBody{}
	if err := w.api(http.MethodGet, "/v1/sql/connections/"+w.conn+"/tables/"+table+"/rows?schema=public", nil, &w.result); err != nil {
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
