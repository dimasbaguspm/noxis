package uat

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	"noxis/test/uat/pgmock"
)

// postgresMockHost is the compose hostname that resolves to the machine running the test process.
const postgresMockHost = "host.docker.internal"

// postgresMockSteps drive the protocol mock: noXis connects to it as if it were a real server.
func (w *world) postgresMockSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a mock postgres connection named "([^"]*)"$`, w.registerMockPostgres)
	sc.Step(`^the mock postgres answers rows with column "([^"]*)" and value "([^"]*)"$`, w.mockPostgresAnswers)
	sc.Step(`^the mock postgres answers with error "([^"]*)"$`, w.mockPostgresFails)
	sc.Step(`^the mock postgres received the query "([^"]*)"$`, w.mockPostgresReceived)
	sc.Step("the mock postgres received a request", w.mockPostgresReceivedAny)
}

// registerMockPostgres starts a mock target and registers it as a connection; simple_protocol keeps the mock's protocol surface small.
func (w *world) registerMockPostgres(ctx context.Context, name string) error {
	mock, err := pgmock.Start()
	if err != nil {
		return fmt.Errorf("start mock postgres: %w", err)
	}
	w.pgMock = mock
	conn := map[string]any{
		"name": name, "driver": "postgres", "host": postgresMockHost, "port": mock.Port(),
		"user": "mock", "password": "mock", "database": "demo",
		"params": map[string]string{"default_query_exec_mode": "simple_protocol"},
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

func (w *world) mockPostgresAnswers(ctx context.Context, column, value string) error {
	if w.pgMock == nil {
		return fmt.Errorf("no mock postgres is running")
	}
	w.pgMock.Answer([]string{column}, []string{value})
	return nil
}

func (w *world) mockPostgresFails(ctx context.Context, message string) error {
	if w.pgMock == nil {
		return fmt.Errorf("no mock postgres is running")
	}
	w.pgMock.Fail(message)
	return nil
}

func (w *world) mockPostgresReceived(ctx context.Context, want string) error {
	if w.pgMock == nil {
		return fmt.Errorf("no mock postgres is running")
	}
	for _, query := range w.pgMock.Queries() {
		if query == want {
			return nil
		}
	}
	return fmt.Errorf("mock postgres did not receive %q (received %v)", want, w.pgMock.Queries())
}

func (w *world) mockPostgresReceivedAny(ctx context.Context) error {
	if w.pgMock == nil {
		return fmt.Errorf("no mock postgres is running")
	}
	if len(w.pgMock.Queries()) == 0 {
		return fmt.Errorf("mock postgres received no request")
	}
	return nil
}
