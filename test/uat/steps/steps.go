// Package steps wires the Gherkin sentences to the API client. Every step talks HTTP to
// the running binary: nothing here opens the state file or a database.
package steps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cucumber/godog"

	"noxis/internal/client"
)

// world is one scenario's state.
type world struct {
	api      *client.Client
	identity client.Identity
	health   client.Health
	err      error
}

// InitializeScenario registers every step against a fresh world.
func InitializeScenario(ctx *godog.ScenarioContext) {
	w := &world{}
	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		w.identity, w.health, w.err = client.Identity{}, client.Health{}, nil
		w.api, w.err = client.New(os.Getenv("UAT_URL"), "")
		return c, w.err
	})

	ctx.Step(`^I hold the (agent|ops) token$`, w.holdToken)
	ctx.Step(`^I ask who I am$`, w.askWhoami)
	ctx.Step(`^I ask who I am without a token$`, w.askWhoamiWithoutToken)
	ctx.Step(`^I ask for the health of the server without a token$`, w.askHealth)
	ctx.Step(`^my name is "([^"]*)"$`, w.assertName)
	ctx.Step(`^my scopes are "([^"]*)"$`, w.assertScopes)
	ctx.Step(`^the request succeeds$`, w.assertSuccess)
	ctx.Step(`^the request fails with status (\d+)$`, w.assertStatus)
	ctx.Step(`^the reported version is "([^"]*)"$`, w.assertVersion)
}

func (w *world) holdToken(name string) error {
	secret := map[string]string{
		"agent": os.Getenv("UAT_AGENT_SECRET"),
		"ops":   os.Getenv("UAT_OPS_SECRET"),
	}[name]
	if secret == "" {
		return fmt.Errorf("no %s secret in the environment", name)
	}
	api, err := client.New(os.Getenv("UAT_URL"), secret)
	w.api = api
	return err
}

func (w *world) askWhoami() error {
	w.identity, w.err = w.api.Whoami(context.Background())
	return nil
}

func (w *world) askWhoamiWithoutToken() error {
	anonymous, err := client.New(os.Getenv("UAT_URL"), "")
	if err != nil {
		return err
	}
	w.identity, w.err = anonymous.Whoami(context.Background())
	return nil
}

func (w *world) askHealth() error {
	w.health, w.err = w.api.Live(context.Background())
	return nil
}

func (w *world) assertName(want string) error {
	if w.err != nil {
		return w.err
	}
	if w.identity.Name != want {
		return fmt.Errorf("name = %q, want %q", w.identity.Name, want)
	}
	return nil
}

func (w *world) assertScopes(want string) error {
	if w.err != nil {
		return w.err
	}
	got := strings.Join(w.identity.Scopes, ", ")
	if got != want {
		return fmt.Errorf("scopes = %q, want %q", got, want)
	}
	return nil
}

func (w *world) assertVersion(want string) error {
	if w.err != nil {
		return w.err
	}
	if w.health.Version != want {
		return fmt.Errorf("version = %q, want %q", w.health.Version, want)
	}
	return nil
}

func (w *world) assertSuccess() error { return w.err }

func (w *world) assertStatus(want int) error {
	if w.err == nil {
		return fmt.Errorf("the request succeeded, want status %d", want)
	}
	var apiErr *client.Error
	if !errors.As(w.err, &apiErr) {
		return w.err
	}
	if apiErr.Status != want {
		return fmt.Errorf("status = %d, want %d", apiErr.Status, want)
	}
	return nil
}
