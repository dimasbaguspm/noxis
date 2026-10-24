package uat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"
	"go.mongodb.org/mongo-driver/v2/bson"

	"noxis/test/uat/mongomock"
)

func (w *world) mongoMockSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a mock mongo connection named "([^"]*)"$`, w.registerMockMongo)
	sc.Step(`^the mock mongo answers documents (.+)$`, w.mockMongoAnswers)
	sc.Step(`^the mock mongo answers with error "([^"]*)"$`, w.mockMongoFails)
	sc.Step(`^the mock mongo received a "([^"]*)" command$`, w.mockMongoReceived)
	sc.Step(`^the mock mongo received a find filter with "([^"]*)" = "([^"]*)"$`, w.mockMongoFilter)
}

// registerMockMongo starts a MongoDB-protocol mock and registers it as a connection.
func (w *world) registerMockMongo(ctx context.Context, name string) error {
	mock, err := mongomock.Start()
	if err != nil {
		return fmt.Errorf("start mock mongo: %w", err)
	}
	w.mongoMock = mock
	conn := map[string]any{
		"name": name, "driver": "mongo", "host": postgresMockHost, "port": mock.Port(),
		"user": "mock", "password": "mock", "database": "admin",
	}
	if err := w.api(http.MethodPost, "/external/v1/engine/mongodb/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register mock connection: %d %s", w.code, w.body)
	}
	w.conn = name
	return nil
}

func (w *world) mockMongoAnswers(ctx context.Context, documents string) error {
	if w.mongoMock == nil {
		return fmt.Errorf("no mock mongo is running")
	}
	var docs []bson.M
	if err := json.Unmarshal([]byte(documents), &docs); err != nil {
		return fmt.Errorf("documents %q: %w", documents, err)
	}
	w.mongoMock.Answer(docs...)
	return nil
}

func (w *world) mockMongoFails(ctx context.Context, message string) error {
	if w.mongoMock == nil {
		return fmt.Errorf("no mock mongo is running")
	}
	w.mongoMock.Fail(message)
	return nil
}

func (w *world) mockMongoReceived(ctx context.Context, command string) error {
	if w.mongoMock == nil {
		return fmt.Errorf("no mock mongo is running")
	}
	for _, got := range w.mongoMock.Commands() {
		if got == command {
			return nil
		}
	}
	return fmt.Errorf("mock mongo did not receive %q (received %v)", command, w.mongoMock.Commands())
}

func (w *world) mockMongoFilter(ctx context.Context, key, value string) error {
	if w.mongoMock == nil {
		return fmt.Errorf("no mock mongo is running")
	}
	for _, filter := range w.mongoMock.Filters() {
		if fmt.Sprint(filter[key]) == value {
			return nil
		}
	}
	return fmt.Errorf("no find filter carried %s=%q (received %v)", key, value, w.mongoMock.Filters())
}
