package uat

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/cucumber/godog"
)

// The MongoDB scenarios drive the /v1/mongodb namespace: register the compose
// server, explore databases and collections with their sizes, run CRUD
// requests with separate filter and value documents, and read stats.
// Documents are seeded with mongosh because noXis has no "seed" endpoint.

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

// mongoInfo is the subset of stats the scenarios assert on.
type mongoInfo struct {
	Database        string `json:"database"`
	Collection      string `json:"collection"`
	Version         string `json:"version"`
	Connections     int64  `json:"connections"`
	Inserted        int64  `json:"inserted"`
	Objects         int64  `json:"dbObjects"`
	CollectionCount int64  `json:"collectionCount"`
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
	sc.Step(`^I read mongo stats for "([^"]*)" and "([^"]*)"$`, w.mongoStats)
	sc.Step("the mongo stats describe the server and the collection", w.mongoStatsDescribe)
}

func (w *world) registerMongo(ctx context.Context, name string) error {
	conn := map[string]any{
		"name": name, "driver": "mongo", "host": "mongo", "port": 27017,
		"user": "root", "password": "noxis", "database": "admin",
	}
	if err := w.api(http.MethodPost, "/v1/mongodb/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register mongo connection: %d", w.code)
	}
	w.conn = name
	return nil
}

func (w *world) mongoAnswers(ctx context.Context) error {
	if err := w.api(http.MethodPost, "/v1/mongodb/connections/"+w.conn+"/test", map[string]string{}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("mongo ping: %d", w.code)
	}
	return nil
}

// mongoSeed inserts documents with mongosh, authenticating as root.
func (w *world) mongoSeed(ctx context.Context, collection, documents string) error {
	script := fmt.Sprintf(
		`db.getSiblingDB("demo").getCollection(%q).insertMany(%s)`,
		collection, documents,
	)
	out, err := exec.Command("docker", "compose", "-f", stackFile, "exec", "-T", "mongo",
		"mongosh", "-u", "root", "-p", "noxis", "--authenticationDatabase", "admin",
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
	path := "/v1/mongodb/connections/" + w.conn + "/databases"
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
	path := fmt.Sprintf("/v1/mongodb/connections/%s/collections?database=%s", w.conn, database)
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
		"op": op, "database": "demo", "collection": collection,
		"filter": filter, "value": value, "limit": 50,
	}
	path := "/v1/mongodb/connections/" + w.conn + "/requests"
	if err := w.api(http.MethodPost, path, body, &w.result); err != nil {
		return err
	}
	// 400 is left to the rejection step to assert; anything else unexpected is
	// a failure right here.
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

// mongoRejected asserts a refused request (an empty filter on a delete, for
// instance) and that the message says why.
func (w *world) mongoRejected(ctx context.Context, fragment string) error {
	if w.code != http.StatusBadRequest {
		return fmt.Errorf("request status %d, want 400", w.code)
	}
	if !strings.Contains(w.body, fragment) {
		return fmt.Errorf("error %q does not mention %q", w.body, fragment)
	}
	return nil
}

func (w *world) mongoStats(ctx context.Context, database, collection string) error {
	w.mongo = mongoInfo{}
	path := fmt.Sprintf("/v1/mongodb/connections/%s/stats?database=%s&collection=%s", w.conn, database, collection)
	if err := w.api(http.MethodGet, path, nil, &w.mongo); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("mongo stats: %d", w.code)
	}
	return nil
}

func (w *world) mongoStatsDescribe(ctx context.Context) error {
	if w.mongo.Version == "" || w.mongo.Database == "" {
		return fmt.Errorf("stats are missing server or database fields: %+v", w.mongo)
	}
	if w.mongo.CollectionCount <= 0 {
		return fmt.Errorf("collection stats are empty: %+v", w.mongo)
	}
	return nil
}
