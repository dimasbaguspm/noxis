// Package steps wires the Gherkin sentences to the API client. Every step talks HTTP to the
// running binary: nothing here opens the state file or a database.
package steps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cucumber/godog"

	"noxis/internal/client"
	"noxis/internal/common/ref"
	"noxis/internal/constants"
)

// world is one scenario's state.
type world struct {
	api      *client.Client
	identity client.Identity
	health   client.Health
	target   client.Target
	targets  []client.Target
	caps     client.Capabilities
	cursor   string
	err      error
}

// InitializeScenario registers every step against a fresh world.
func InitializeScenario(ctx *godog.ScenarioContext) {
	w := &world{}
	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		w.identity, w.health = client.Identity{}, client.Health{}
		w.target, w.targets, w.cursor, w.err = client.Target{}, nil, "", nil
		w.caps = client.Capabilities{}
		w.api, w.err = client.New(os.Getenv("UAT_URL"), "")
		return c, w.err
	})

	// identity
	ctx.Step(`^I hold the (agent|ops) token$`, w.holdToken)
	ctx.Step(`^I ask who I am$`, w.askWhoami)
	ctx.Step(`^I ask who I am without a token$`, w.askWhoamiWithoutToken)
	ctx.Step(`^I ask for the health of the server without a token$`, w.askHealth)
	ctx.Step(`^my name is "([^"]*)"$`, w.assertName)
	ctx.Step(`^my scopes are "([^"]*)"$`, w.assertScopes)
	ctx.Step(`^the reported version is "([^"]*)"$`, w.assertVersion)

	// targets
	ctx.Step(`^I register a target named "([^"]*)"$`, w.registerTarget)
	ctx.Step(`^I register three targets$`, w.registerThree)
	ctx.Step(`^the target reference is a reference$`, w.assertTargetRef)
	ctx.Step(`^the password is stored$`, w.assertPasswordStored)
	ctx.Step(`^the response carries no password$`, w.assertNoPasswordOnWire)
	ctx.Step(`^I list targets two at a time$`, w.listTwo)
	ctx.Step(`^I follow the cursor$`, w.followCursor)
	ctx.Step(`^I see (\d+) targets?$`, w.assertCount)
	ctx.Step(`^there is a next cursor$`, w.assertHasCursor)
	ctx.Step(`^there is no next cursor$`, w.assertNoCursor)
	ctx.Step(`^I read the target "([^"]*)"$`, w.readTarget)
	ctx.Step(`^I archive the target I registered$`, w.archiveTarget)

	// probe
	ctx.Step(`^I register a target named "([^"]*)" on port (\d+)$`, w.registerTargetOnPort)
	ctx.Step(`^I ask for the capabilities of the target I registered$`, w.askCapabilities)
	ctx.Step(`^I probe the target I registered$`, w.probeTarget)
	ctx.Step(`^the capabilities name the "([^"]*)" engine$`, w.assertFlavor)
	ctx.Step(`^the capabilities say it has not been probed$`, w.assertNotProbed)
	ctx.Step(`^the target reports probe status "([^"]*)"$`, w.assertProbeStatus)

	// shared assertions
	ctx.Step(`^the request succeeds$`, w.assertSuccess)
	ctx.Step(`^the request succeeds with status (\d+)$`, w.assertSuccessStatus)
	ctx.Step(`^the request fails with status (\d+)$`, w.assertStatus)
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
	if got := strings.Join(w.identity.Scopes, ", "); got != want {
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

// registerTarget registers one target with a password, so the sealing path is exercised.
func (w *world) registerTarget(name string) error {
	w.target, w.err = w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:     name,
		Driver:   constants.DriverMySQL,
		Host:     "127.0.0.1",
		Port:     constants.DefaultMySQLPort,
		Database: "uat",
		Username: "uat",
		Password: "uat-password",
		TLSMode:  constants.TLSPreferred,
	})
	return nil
}

func (w *world) registerThree() error {
	for _, name := range []string{"uat-paged-a", "uat-paged-b", "uat-paged-c"} {
		if err := w.registerTarget(name); err != nil {
			return err
		}
		if w.err != nil {
			return w.err
		}
	}
	return nil
}

func (w *world) assertTargetRef() error {
	if w.err != nil {
		return w.err
	}
	if !ref.Valid(w.target.Ref) {
		return fmt.Errorf("ref = %q, not a reference", w.target.Ref)
	}
	return nil
}

func (w *world) assertPasswordStored() error {
	if w.err != nil {
		return w.err
	}
	if !w.target.HasPassword {
		return errors.New("the target reports no password, so it was not sealed")
	}
	return nil
}

// assertNoPasswordOnWire reads the target again with a bare request and checks the body,
// because a typed client cannot prove what the server actually sent.
func (w *world) assertNoPasswordOnWire() error {
	if w.err != nil {
		return w.err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		os.Getenv("UAT_URL")+constants.PathTargets+"/"+w.target.Ref, nil)
	if err != nil {
		return err
	}
	request.Header.Set(constants.HeaderAuthorization, constants.SchemeBearer+" "+os.Getenv("UAT_OPS_SECRET"))
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	for _, leaked := range []string{"password_cipher", `"password":`, "uat-password", "v1:"} {
		if strings.Contains(string(body), leaked) {
			return fmt.Errorf("the response carries %q", leaked)
		}
	}
	return nil
}

// listTwo pages only its own three targets, because scenarios share one server and the
// state file grows across them.
func (w *world) listTwo() error {
	page, err := w.api.ListTargets(context.Background(), client.ListTargetsInput{Limit: 2, Query: "uat-paged"})
	w.targets, w.cursor, w.err = page.Items, page.NextCursor, err
	return nil
}

func (w *world) followCursor() error {
	if w.err != nil {
		return w.err
	}
	page, err := w.api.ListTargets(context.Background(), client.ListTargetsInput{Limit: 2, Query: "uat-paged", Cursor: w.cursor})
	w.targets, w.cursor, w.err = page.Items, page.NextCursor, err
	return nil
}

func (w *world) assertCount(want int) error {
	if w.err != nil {
		return w.err
	}
	if len(w.targets) != want {
		return fmt.Errorf("saw %d targets, want %d", len(w.targets), want)
	}
	return nil
}

func (w *world) assertHasCursor() error {
	if w.err != nil {
		return w.err
	}
	if w.cursor == "" {
		return errors.New("no next cursor on a full page")
	}
	return nil
}

func (w *world) assertNoCursor() error {
	if w.err != nil {
		return w.err
	}
	if w.cursor != "" {
		return fmt.Errorf("next cursor %q on the last page", w.cursor)
	}
	return nil
}

func (w *world) readTarget(value string) error {
	_, w.err = w.api.GetTarget(context.Background(), value)
	return nil
}

func (w *world) archiveTarget() error {
	if w.err != nil {
		return w.err
	}
	_, w.err = w.api.ArchiveTarget(context.Background(), w.target.Ref)
	return nil
}

func (w *world) assertSuccess() error { return w.err }

func (w *world) assertSuccessStatus(want int) error {
	if w.err != nil {
		return w.err
	}
	if got := w.api.LastStatus(); got != want {
		return fmt.Errorf("status = %d, want %d", got, want)
	}
	return nil
}

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

// registerTargetOnPort registers a target on a chosen port, so a probe can be pointed at a
// socket nothing is listening on and fail in a way the test controls.
func (w *world) registerTargetOnPort(name string, port int) error {
	w.target, w.err = w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:     name,
		Driver:   constants.DriverMySQL,
		Host:     "127.0.0.1",
		Port:     port,
		Database: "uat",
		Username: "uat",
		Password: "uat-password",
		TLSMode:  constants.TLSPreferred,
	})
	return nil
}

func (w *world) askCapabilities() error {
	w.caps, w.err = w.api.TargetCapabilities(context.Background(), w.target.Ref)
	return nil
}

func (w *world) probeTarget() error {
	result, err := w.api.ProbeTarget(context.Background(), w.target.Ref)
	w.caps, w.err = result.Capabilities, err
	return nil
}

func (w *world) assertFlavor(want string) error {
	if w.err != nil {
		return w.err
	}
	if w.caps.Flavor != want {
		return fmt.Errorf("flavor = %q, want %q (notes: %v)", w.caps.Flavor, want, w.caps.Notes)
	}
	return nil
}

func (w *world) assertNotProbed() error {
	if w.err != nil {
		return w.err
	}
	if w.caps.Version != "" {
		return fmt.Errorf("capabilities report version %q for an unprobed target", w.caps.Version)
	}
	for _, note := range w.caps.Notes {
		if strings.Contains(note, "not probed yet") {
			return nil
		}
	}
	return fmt.Errorf("capabilities do not say the target has not been probed: %v", w.caps.Notes)
}

func (w *world) assertProbeStatus(want string) error {
	stored, err := w.api.GetTarget(context.Background(), w.target.Ref)
	if err != nil {
		return err
	}
	if stored.LastProbeStatus != want {
		return fmt.Errorf("probe status = %q, want %q", stored.LastProbeStatus, want)
	}
	if stored.LastProbeAt == nil {
		return errors.New("a probe that ran left no timestamp")
	}
	return nil
}
