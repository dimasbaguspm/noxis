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
	sc.Step(`^the current connection matches snapshot "([^"]*)"$`, w.connectionSnapshot)
	sc.Step("the instances list contains the current connection", w.instancesListContainsCurrent)
	sc.Step("I open the current instance", w.openCurrentInstance)
	sc.Step("I rotate the instance password", w.rotateInstancePassword)
	sc.Step("the current connection still hides its password", w.connectionHidesPassword)
	sc.Step("changing the connection's host is refused", w.identityChangeRefused)
	sc.Step("I delete the current connection", w.deleteCurrentConnection)
	sc.Step("the connection is gone", w.connectionGone)
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
	if err := w.api(http.MethodPost, enginePathOf(t.Driver, "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register %s connection: %d", key, w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, key
	return nil
}

func (w *world) testConnection(ctx context.Context) error {
	if err := w.api(http.MethodPost, w.enginePath("/connections/"+w.conn+"/test"), map[string]string{}, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("test connection: %d", w.code)
	}
	return nil
}

func (w *world) listRedacted(ctx context.Context) error {
	code, body, err := w.raw(w.enginePath("/connections"))
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

// instance is one row of the cross-family instances list.
type instance struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// instancesListContainsCurrent proves the connection is addressable by its stable id.
func (w *world) instancesListContainsCurrent(ctx context.Context) error {
	var out struct {
		Instances []instance `json:"instances"`
	}
	if err := w.api(http.MethodGet, "/external/v1/instances", nil, &out); err != nil {
		return err
	}
	for _, in := range out.Instances {
		if in.Name == w.conn {
			w.instanceID = in.ID
			return nil
		}
	}
	return fmt.Errorf("connection %q not in %d instance(s)", w.conn, len(out.Instances))
}

func (w *world) openCurrentInstance(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/external/v1/instances/"+w.instanceID, nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, `"`+w.conn+`"`) {
		return fmt.Errorf("instance detail: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) rotateInstancePassword(ctx context.Context) error {
	body := map[string]string{"password": "rotated-uat-secret"}
	if err := w.api(http.MethodPost, "/external/v1/instances/"+w.instanceID+"/password", body, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("rotate password: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) connectionHidesPassword(ctx context.Context) error {
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn), nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || strings.Contains(strings.ToLower(w.body), `"password"`) {
		return fmt.Errorf("connection detail: %d %s", w.code, w.body)
	}
	return nil
}

// identityChangeRefused proves the identity fields are locked after registration.
func (w *world) identityChangeRefused(ctx context.Context) error {
	body := map[string]any{
		"name": w.conn, "driver": "postgres", "host": "elsewhere", "port": 5432,
		"user": "postgres", "database": "demo",
	}
	if err := w.api(http.MethodPost, w.enginePath("/connections"), body, nil); err != nil {
		return err
	}
	if w.code == http.StatusOK {
		return fmt.Errorf("changing the host must be refused: %s", w.body)
	}
	return nil
}

func (w *world) deleteCurrentConnection(ctx context.Context) error {
	if err := w.api(http.MethodDelete, w.enginePath("/connections/"+w.conn), nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK && w.code != http.StatusNoContent {
		return fmt.Errorf("delete connection: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) connectionGone(ctx context.Context) error {
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn), nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusNotFound {
		return fmt.Errorf("deleted connection: %d %s, want 404", w.code, w.body)
	}
	return nil
}

// connectionSnapshot compares the current connection's detail response with a golden.
func (w *world) connectionSnapshot(ctx context.Context, name string) error {
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn), nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("connection detail: %d %s", w.code, w.body)
	}
	return w.snapshots.Match(name, w.resp.Body)
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
