package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// connectionSteps cover registering a target and the connection-level assertions.
func (w *world) connectionSteps(sc *godog.ScenarioContext) {
	sc.Step(`^connection "([^"]*)" is registered$`, w.registerConnection)
	sc.Step(`^connection "([^"]*)" is registered for driver "([^"]*)"$`, w.registerConnectionDriver)
	sc.Step(`^a fresh connection "([^"]*)" is registered$`, w.registerFreshConnection)
	sc.Step("the connection test passes", w.testConnection)
	sc.Step("the connection list hides the password", w.listRedacted)
	sc.Step(`^the operator console is served at "/"$`, w.consoleServed)
}

// registerConnection upserts the default SQL target as a noxis connection.
func (w *world) registerConnection(ctx context.Context, name string) error {
	return w.registerSQL(ctx, name, "postgres")
}

// registerFreshConnection registers a uniquely named connection, so a scenario that leaves state behind stays repeatable.
func (w *world) registerFreshConnection(ctx context.Context, prefix string) error {
	return w.registerSQL(ctx, fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()), "postgres")
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
	// Assert the password key is absent; substring checks can match unrelated host names or subjects.
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
