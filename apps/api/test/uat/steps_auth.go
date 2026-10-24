package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

func (w *world) authSteps(sc *godog.ScenarioContext) {
	sc.Step(`^the driver catalog lists "([^"]*)"$`, w.driverCatalogLists)
	sc.Step(`^the level catalog lists "([^"]*)"$`, w.levelCatalogLists)
	sc.Step("the auth config is readable", w.authConfigReadable)
	sc.Step(`^I log in with email "([^"]*)" and password "([^"]*)"$`, w.loginWith)
	sc.Step("the login succeeds", w.loginSucceeds)
	sc.Step("the login is refused", w.loginRefused)
	sc.Step("my profile names the root account", w.profileNamesRoot)
	sc.Step(`^a request to "([^"]*)" without a token is refused$`, w.requestWithoutToken)
}

func (w *world) driverCatalogLists(ctx context.Context, driver string) error {
	var out []map[string]any
	if err := w.api(http.MethodGet, "/external/v1/config/drivers", nil, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("drivers: %d", w.code)
	}
	if !strings.Contains(w.body, `"`+driver+`"`) {
		return fmt.Errorf("driver %q not in %s", driver, w.body)
	}
	return nil
}

func (w *world) levelCatalogLists(ctx context.Context, level string) error {
	if err := w.api(http.MethodGet, "/external/v1/config/levels", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("levels: %d", w.code)
	}
	if !strings.Contains(w.body, `"`+level+`"`) {
		return fmt.Errorf("level %q not in %s", level, w.body)
	}
	return nil
}

func (w *world) authConfigReadable(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/external/v1/auth/config", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("auth config: %d", w.code)
	}
	if !strings.Contains(w.body, `"local"`) {
		return fmt.Errorf("auth config = %s", w.body)
	}
	return nil
}

// loginWith posts credentials and keeps the response for the next assertion; a successful token is not installed, so the suite's session survives.
func (w *world) loginWith(ctx context.Context, email, password string) error {
	return w.api(http.MethodPost, "/external/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
}

func (w *world) loginSucceeds(ctx context.Context) error {
	if w.code != http.StatusOK || !strings.Contains(w.body, `"token"`) {
		return fmt.Errorf("login: %d %s", w.code, w.body)
	}
	if strings.Contains(w.body, `"password"`) {
		return fmt.Errorf("login response carries a password: %s", w.body)
	}
	return nil
}

func (w *world) loginRefused(ctx context.Context) error {
	if w.code == http.StatusOK {
		return fmt.Errorf("login should have been refused: %s", w.body)
	}
	return nil
}

func (w *world) profileNamesRoot(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/external/v1/me", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, adminEmail) {
		return fmt.Errorf("me: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) requestWithoutToken(ctx context.Context, path string) error {
	w.headers["Authorization"] = ""
	if err := w.api(http.MethodGet, path, nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusUnauthorized {
		return fmt.Errorf("%s without a token: %d, want 401", path, w.code)
	}
	return nil
}
