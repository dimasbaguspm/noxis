package uat

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/cucumber/godog"
)

// MongoDB scenarios browse and review writes; mongosh seeds documents because noXis has no seed endpoint.

// mongoDatabase is one row of the explorer.
type mongoDatabase struct {
	Name        string `json:"name"`
	SizeBytes   int64  `json:"sizeBytes"`
	Collections int64  `json:"collections"`
	Objects     int64  `json:"objects"`
}

// mongoCollection is one row of a database's collections.
type mongoCollection struct {
	Name      string `json:"name"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"sizeBytes"`
}

func (w *world) mongoSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a mongo connection named "([^"]*)"$`, w.registerMongo)
	sc.Step("the mongo server answers", w.mongoAnswers)
	sc.Step(`^the mongo database "([^"]*)" exists$`, w.mongoDatabaseExists)
	sc.Step(`^the mongo database "([^"]*)" holds collection "([^"]*)"$`, w.mongoCollectionExists)
	sc.Step(`^I seed mongo collection "([^"]*)" with (.+)$`, w.mongoSeed)
	sc.Step(`^I run a mongo "([^"]*)" request on "([^"]*)" with filter (.+?) and value (.+)$`, w.mongoRequest)
	sc.Step(`^the mongo result has (\d+) row\(s\)$`, w.mongoRowCount)
	sc.Step(`^the mongo result reports (\d+) affected$`, w.mongoAffected)
	sc.Step(`^the mongo request is rejected with "([^"]*)"$`, w.mongoRejected)
	sc.Step(`^I request a mongo "([^"]*)" on "([^"]*)" with filter (.+?) and value (.+)$`, w.mongoRequestChange)
	sc.Step("the mongo request is validated", w.mongoRequestValidated)
	sc.Step(`^the mongo request estimates (\d+) affected$`, w.mongoRequestEstimate)
	sc.Step("the mongo request was not recorded", w.mongoRequestNotRecorded)
	sc.Step("I approve the mongo request", w.mongoRequestApprove)
	sc.Step("I apply the mongo request", w.mongoRequestApply)
	sc.Step(`^the mongo request is "([^"]*)"$`, w.mongoRequestStatus)
	sc.Step("a read op is refused by the request queue", w.mongoReadRefused)
	sc.Step(`^the mongo request checks mention "([^"]*)"$`, w.mongoChecksMention)
}

// mongoRequestChange submits a write as a request for review, the way the console does.
func (w *world) mongoRequestChange(ctx context.Context, op, collection, filter, value string) error {
	w.change = changeBody{}
	w.mongoQueueSize = w.mongoQueueLen()
	body := map[string]any{
		"connection": w.conn, "notes": "uat " + op, "op": op, "database": targets["mongo"].Database,
		"collection": collection, "filter": filter, "value": value, "many": false,
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit mongo request: %d", w.code)
	}
	return nil
}

func (w *world) mongoRequestValidated(ctx context.Context) error {
	if w.change.Status != "validated" {
		return fmt.Errorf("request status %q want validated (checks: %+v)", w.change.Status, w.change.Checks)
	}
	return nil
}

func (w *world) mongoRequestEstimate(ctx context.Context, want int) error {
	if w.change.EstAffected != int64(want) {
		return fmt.Errorf("estimate %d want %d", w.change.EstAffected, want)
	}
	return nil
}

// mongoChecksMention asserts a validation check says why a request was refused (a delete with no filter, an unknown update operator).
func (w *world) mongoChecksMention(ctx context.Context, fragment string) error {
	for _, check := range w.change.Checks {
		if strings.Contains(check.Error, fragment) {
			return nil
		}
	}
	return fmt.Errorf("no check mentions %q (checks: %+v)", fragment, w.change.Checks)
}

// mongoQueueLen reads how many requests are already in the queue.
func (w *world) mongoQueueLen() int {
	var out struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, w.enginePath("/changes"), nil, &out); err != nil {
		return -1
	}
	return len(out.Changes)
}

// mongoRequestNotRecorded proves a failed validation is a rejected proposal: it is returned with its checks and never enters the queue.
func (w *world) mongoRequestNotRecorded(ctx context.Context) error {
	if w.change.ID != "" {
		return fmt.Errorf("failed request was recorded with id %s", w.change.ID)
	}
	var out struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, w.enginePath("/changes"), nil, &out); err != nil {
		return err
	}
	if len(out.Changes) != w.mongoQueueSize {
		return fmt.Errorf("queue grew to %d, want %d", len(out.Changes), w.mongoQueueSize)
	}
	return nil
}

func (w *world) mongoRequestApprove(ctx context.Context) error {
	if err := w.api(http.MethodPost, w.enginePath("/changes/"+w.change.ID+"/approve"), map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("approve: %d", w.code)
	}
	return nil
}

func (w *world) mongoRequestApply(ctx context.Context) error {
	if err := w.api(http.MethodPost, w.enginePath("/changes/"+w.change.ID+"/apply"), map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("apply: %d", w.code)
	}
	return nil
}

func (w *world) mongoRequestStatus(ctx context.Context, want string) error {
	if w.change.Status != want {
		return fmt.Errorf("request status %q want %q (checks: %+v)", w.change.Status, want, w.change.Checks)
	}
	return nil
}

// mongoReadRefused proves a read cannot be queued as a request.
func (w *world) mongoReadRefused(ctx context.Context) error {
	body := map[string]any{
		"connection": w.conn, "notes": "read", "op": "find", "database": targets["mongo"].Database, "collection": "items",
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.change.Status != "failed" {
		return fmt.Errorf("read request status %q want failed", w.change.Status)
	}
	return nil
}

func (w *world) registerMongo(ctx context.Context, name string) error {
	t := targets["mongo"]
	conn := map[string]any{
		"name": name, "driver": t.Driver, "host": t.Host, "port": t.Port,
		"user": t.User, "password": t.Password, "database": "admin",
	}
	if err := w.api(http.MethodPost, enginePathOf(t.Driver, "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register mongo connection: %d", w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, "mongo"
	return nil
}

func (w *world) mongoAnswers(ctx context.Context) error {
	if err := w.api(http.MethodPost, w.enginePath("/connections/"+w.conn+"/test"), map[string]string{}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("mongo ping: %d", w.code)
	}
	return nil
}

// mongoSeed inserts documents with mongosh, authenticating as root.
func (w *world) mongoSeed(ctx context.Context, collection, documents string) error {
	t := targets["mongo"]
	script := fmt.Sprintf(
		`db.getSiblingDB(%q).getCollection(%q).insertMany(%s)`,
		t.Database, collection, documents,
	)
	out, err := exec.Command("docker", "compose", "-f", stackFile, "exec", "-T", t.Host,
		"mongosh", "-u", t.User, "-p", t.Password, "--authenticationDatabase", "admin",
		"--quiet", "--eval", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("seed %s: %w\n%s", collection, err, out)
	}
	return nil
}

func (w *world) mongoDatabaseExists(ctx context.Context, database string) error {
	var out struct {
		Databases []mongoDatabase `json:"databases"`
	}
	path := w.enginePath("/connections/" + w.conn + "/databases")
	if err := w.api(http.MethodGet, path, nil, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("list databases: %d", w.code)
	}
	for _, db := range out.Databases {
		if db.Name == database {
			w.database = db
			return nil
		}
	}
	return fmt.Errorf("database %q not in %d listed", database, len(out.Databases))
}

func (w *world) mongoCollectionExists(ctx context.Context, database, collection string) error {
	var out struct {
		Collections []mongoCollection `json:"collections"`
	}
	path := fmt.Sprintf(w.enginePath("/connections/%s/collections?database=%s"), w.conn, database)
	if err := w.api(http.MethodGet, path, nil, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("list collections: %d", w.code)
	}
	for _, c := range out.Collections {
		if c.Name == collection {
			w.collection = c
			return nil
		}
	}
	return fmt.Errorf("collection %q not in %d listed", collection, len(out.Collections))
}

func (w *world) mongoRequest(ctx context.Context, op, collection, filter, value string) error {
	w.result = resultBody{}
	body := map[string]any{
		"op": op, "database": targets["mongo"].Database, "collection": collection,
		"filter": filter, "value": value, "limit": 50,
	}
	path := w.enginePath("/connections/" + w.conn + "/requests")
	if err := w.api(http.MethodPost, path, body, &w.result); err != nil {
		return err
	}
	// 400 is left to the rejection step to assert; anything else unexpected is a failure right here.
	if w.code != http.StatusOK && w.code != http.StatusBadRequest {
		return fmt.Errorf("mongo %s: %d", op, w.code)
	}
	return nil
}

func (w *world) mongoRowCount(ctx context.Context, want int) error {
	if len(w.result.Rows) != want {
		return fmt.Errorf("mongo rows=%d want %d (%v)", len(w.result.Rows), want, w.result.Rows)
	}
	return nil
}

func (w *world) mongoAffected(ctx context.Context, want int) error {
	if w.result.Affected != int64(want) {
		return fmt.Errorf("mongo affected=%d want %d (%s)", w.result.Affected, want, w.result.Message)
	}
	return nil
}

// mongoRejected asserts a refused request (an empty filter on a delete, for instance) and that the message says why.
func (w *world) mongoRejected(ctx context.Context, fragment string) error {
	if w.code != http.StatusBadRequest {
		return fmt.Errorf("request status %d, want 400", w.code)
	}
	if !strings.Contains(w.body, fragment) {
		return fmt.Errorf("error %q does not mention %q", w.body, fragment)
	}
	return nil
}
