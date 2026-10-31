package uat

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/cucumber/godog"
)

// CouchDB scenarios browse a database of documents and review one write. curl inside the container seeds, because noxis
// has no endpoint that writes a document.

// couchResult is a read answer: the documents the engine returned, as JSON.
type couchResult struct {
	Documents []string `json:"documents"`
}

func (w *world) couchSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a couchdb connection named "([^"]*)"$`, w.registerCouch)
	sc.Step(`^the couchdb database "([^"]*)" exists$`, w.couchDatabase)
	sc.Step(`^I seed couchdb document "([^"]*)" with (.+)$`, w.couchSeed)
	sc.Step(`^the couchdb database list holds "([^"]*)"$`, w.couchDatabaseListed)
	sc.Step(`^I run a couchdb find on "([^"]*)" with selector (.+)$`, w.couchFind)
	sc.Step(`^the couchdb result has (\d+) document\(s\)$`, w.couchDocumentCount)
	sc.Step(`^I request a couchdb "([^"]*)" on "([^"]*)" with value (.+)$`, w.couchRequestChange)
	sc.Step(`^the couchdb request is validated in a scratch database$`, w.couchRequestValidated)
}

// registerCouch registers the compose couchdb service with the admin account its image declares.
func (w *world) registerCouch(ctx context.Context, name string) error {
	t := targets["couchdb"]
	conn := map[string]any{
		"name": name, "driver": t.Driver, "host": t.Host, "port": t.Port,
		"user": t.User, "password": t.Password, "database": t.Database,
	}
	if err := w.api(http.MethodPost, enginePathOf(t.Driver, "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register couchdb connection: %d", w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, "couchdb"
	return nil
}

// couchDatabase creates the database the scenario reads. A second run finds it and moves on, so the suite is repeatable
// against a kept stack.
func (w *world) couchDatabase(ctx context.Context, name string) error {
	return couchCurl("-X", "PUT", "http://localhost:5984/"+name)
}

// couchSeed writes one document straight into the database.
func (w *world) couchSeed(ctx context.Context, id, body string) error {
	return couchCurl("-X", "PUT", "-H", "Content-Type: application/json", "-d", body,
		"http://localhost:5984/"+targets["couchdb"].Database+"/"+id)
}

// couchCurl runs curl inside the couchdb container, which is where the admin credentials already live.
func couchCurl(args ...string) error {
	full := append([]string{"compose", "-f", stackFile, "--profile", "all", "exec", "-T", "couchdb",
		"curl", "-sS", "-u", "admin:noxis"}, args...)
	out, err := exec.Command("docker", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("couchdb %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

// couchDatabaseListed proves the console's database list reaches the server.
func (w *world) couchDatabaseListed(ctx context.Context, want string) error {
	var out struct {
		Databases []struct {
			Name string `json:"name"`
		} `json:"databases"`
	}
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn+"/databases"), nil, &out); err != nil {
		return err
	}
	for _, database := range out.Databases {
		if database.Name == want {
			return nil
		}
	}
	return fmt.Errorf("database list %+v lacks %q", out.Databases, want)
}

// couchFind runs a Mango selector through the console's read route.
func (w *world) couchFind(ctx context.Context, database, selector string) error {
	body := map[string]any{
		"op": "find", "database": database, "collection": "_all_docs", "filter": selector,
	}
	var out couchResult
	if err := w.api(http.MethodPost, w.enginePath("/connections/"+w.conn+"/requests"), body, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("couchdb find: %d", w.code)
	}
	w.couch = out
	return nil
}

func (w *world) couchDocumentCount(ctx context.Context, want int) error {
	if len(w.couch.Documents) != want {
		return fmt.Errorf("couchdb returned %d document(s), want %d", len(w.couch.Documents), want)
	}
	return nil
}

// couchRequestChange submits a write as a request for review, the way the console does.
func (w *world) couchRequestChange(ctx context.Context, op, database, value string) error {
	w.change = changeBody{}
	body := map[string]any{
		"connection": w.conn, "notes": "uat " + op, "op": op, "database": database,
		"collection": "_all_docs", "value": value, "many": false,
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit couchdb request: %d", w.code)
	}
	return nil
}

// couchRequestValidated proves the write was proved in a throwaway database before anyone approved it.
func (w *world) couchRequestValidated(ctx context.Context) error {
	if w.change.Status != "validated" {
		return fmt.Errorf("request status %q want validated (checks: %+v)", w.change.Status, w.change.Checks)
	}
	if !strings.HasPrefix(w.change.ScratchSchema, "noxis_verify_") {
		return fmt.Errorf("request was not proved in a scratch database: %q", w.change.ScratchSchema)
	}
	return nil
}
