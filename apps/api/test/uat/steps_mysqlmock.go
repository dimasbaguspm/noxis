package uat

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	"noxis/test/uat/mysqlmock"
)

func (w *world) mysqlMockSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a mock (mysql|mariadb) connection named "([^"]*)"$`, w.registerMockMySQL)
	sc.Step(`^the mock mysql answers rows with column "([^"]*)" and value "([^"]*)"$`, w.mockMySQLAnswers)
	sc.Step(`^the mock mysql answers with error "([^"]*)"$`, w.mockMySQLFails)
	sc.Step(`^the mock mysql received the query "([^"]*)"$`, w.mockMySQLReceived)
}

// registerMockMySQL starts a MySQL-protocol mock and registers it under the requested driver name.
func (w *world) registerMockMySQL(ctx context.Context, driver, name string) error {
	mock, err := mysqlmock.Start()
	if err != nil {
		return fmt.Errorf("start mock mysql: %w", err)
	}
	w.myMock = mock
	conn := map[string]any{
		"name": name, "driver": driver, "host": postgresMockHost, "port": mock.Port(),
		"user": "mock", "password": "mock", "database": "demo",
	}
	if err := w.api(http.MethodPost, "/external/v1/engine/sql/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register mock connection: %d %s", w.code, w.body)
	}
	w.conn = name
	return nil
}

func (w *world) mockMySQLAnswers(ctx context.Context, column, value string) error {
	if w.myMock == nil {
		return fmt.Errorf("no mock mysql is running")
	}
	w.myMock.Answer([]string{column}, []string{value})
	return nil
}

func (w *world) mockMySQLFails(ctx context.Context, message string) error {
	if w.myMock == nil {
		return fmt.Errorf("no mock mysql is running")
	}
	w.myMock.Fail(message)
	return nil
}

func (w *world) mockMySQLReceived(ctx context.Context, want string) error {
	if w.myMock == nil {
		return fmt.Errorf("no mock mysql is running")
	}
	for _, query := range w.myMock.Queries() {
		if query == want {
			return nil
		}
	}
	return fmt.Errorf("mock mysql did not receive %q (received %v)", want, w.myMock.Queries())
}
