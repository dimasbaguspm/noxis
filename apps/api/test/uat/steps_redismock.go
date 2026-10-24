package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"

	"noxis/test/uat/redismock"
)

func (w *world) redisMockSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a mock (redis|valkey) connection named "([^"]*)"$`, w.registerMockRedis)
	sc.Step(`^the mock cache holds the keys "([^"]*)" with value "([^"]*)"$`, w.mockCacheAnswer)
	sc.Step(`^the mock cache answers with error "([^"]*)"$`, w.mockCacheFails)
	sc.Step(`^the mock cache received the command "([^"]*)"$`, w.mockCacheReceived)
}

// registerMockRedis starts a RESP mock and registers it under the requested driver name.
func (w *world) registerMockRedis(ctx context.Context, driver, name string) error {
	mock, err := redismock.Start()
	if err != nil {
		return fmt.Errorf("start mock cache: %w", err)
	}
	w.redisMock = mock
	conn := map[string]any{
		"name": name, "driver": driver, "host": postgresMockHost, "port": mock.Port(), "database": "0",
	}
	if err := w.api(http.MethodPost, "/external/v1/engine/memcache/connections", conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register mock connection: %d %s", w.code, w.body)
	}
	w.conn = name
	return nil
}

func (w *world) mockCacheAnswer(ctx context.Context, keys, value string) error {
	if w.redisMock == nil {
		return fmt.Errorf("no mock cache is running")
	}
	w.redisMock.Answer(strings.Split(keys, ","), value)
	return nil
}

func (w *world) mockCacheFails(ctx context.Context, message string) error {
	if w.redisMock == nil {
		return fmt.Errorf("no mock cache is running")
	}
	w.redisMock.Fail(message)
	return nil
}

func (w *world) mockCacheReceived(ctx context.Context, verb string) error {
	if w.redisMock == nil {
		return fmt.Errorf("no mock cache is running")
	}
	for _, command := range w.redisMock.Commands() {
		if strings.EqualFold(strings.SplitN(command, " ", 2)[0], verb) {
			return nil
		}
	}
	return fmt.Errorf("mock cache did not receive %q (received %v)", verb, w.redisMock.Commands())
}
