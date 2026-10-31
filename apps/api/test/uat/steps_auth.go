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

// signIn mints a session and installs it. An account holds one token at a time, so every successful sign-in rotates the
// previous one away: without installing it here, every later scenario in the run would present a dead token.
func (w *world) signIn(email, password string) (string, error) {
	var session struct {
		Token string `json:"token"`
	}
	if err := w.api(http.MethodPost, "/external/v1/auth/login",
		map[string]string{"email": email, "password": password}, &session); err != nil {
		return "", err
	}
	if w.code != http.StatusOK || session.Token == "" {
		return "", fmt.Errorf("sign in %s: %d %s", email, w.code, w.body)
	}
	token = session.Token
	w.client.Token = session.Token
	// The override must name the new bearer: an empty override removes the header rather than falling back to the client.
	w.headers["Authorization"] = "Bearer " + session.Token
	return session.Token, nil
}

// loginWith posts credentials and keeps the response for the next assertion. A refusal is the answer under test, so it
// is not an error here; a success installs the session it mints.
func (w *world) loginWith(ctx context.Context, email, password string) error {
	var session struct {
		Token string `json:"token"`
	}
	if err := w.api(http.MethodPost, "/external/v1/auth/login",
		map[string]string{"email": email, "password": password}, &session); err != nil {
		return err
	}
	if w.code == http.StatusOK && session.Token != "" {
		token = session.Token
		w.client.Token = session.Token
		w.headers["Authorization"] = "Bearer " + session.Token
	}
	return nil
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

// routeSteps are the assertions the per-endpoint files need beyond the shared vocabulary. Every expression is anchored:
// an unanchored one matches longer step text and makes the suite ambiguous.
func (w *world) routeSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I request "([^"]*)"$`, w.requestPath)
	sc.Step(`^a (GET|POST|PUT|PATCH|DELETE) to "([^"]*)" without a token is refused$`, w.requestMethodWithoutToken)
	sc.Step(`^the health endpoint reports ok$`, w.healthReportsOK)
	sc.Step(`^the auth config answers without a token$`, w.authConfigPublic)
	sc.Step(`^the member directory lists the root account$`, w.memberDirectoryListsRoot)
	sc.Step(`^the driver catalog does not list "([^"]*)"$`, w.driverCatalogMisses)
	sc.Step(`^I change my password to "([^"]*)" and back$`, w.changePasswordAndBack)
	sc.Step(`^changing my password with the wrong old password is refused$`, w.changePasswordRefused)
	sc.Step(`^I sign out and sign in again$`, w.signOutAndBack)
}

// requestPath sends a plain GET and keeps the response, so a file can assert a status the shared steps do not name.
func (w *world) requestPath(ctx context.Context, path string) error {
	return w.api(http.MethodGet, path, nil, nil)
}

// requestMethodWithoutToken proves a route refuses an anonymous caller with its own method, not just with GET.
func (w *world) requestMethodWithoutToken(ctx context.Context, method, path string) error {
	w.headers["Authorization"] = ""
	if err := w.api(method, path, nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusUnauthorized {
		return fmt.Errorf("%s %s without a token: %d, want 401", method, path, w.code)
	}
	return nil
}

func (w *world) healthReportsOK(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/healthz", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("healthz: %d", w.code)
	}
	return nil
}

// authConfigPublic proves the console can read its sign-in options before it holds a token.
func (w *world) authConfigPublic(ctx context.Context) error {
	w.headers["Authorization"] = ""
	if err := w.api(http.MethodGet, "/external/v1/auth/config", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, `"local"`) {
		return fmt.Errorf("auth config without a token: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) memberDirectoryListsRoot(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/external/v1/config/members", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, adminEmail) {
		return fmt.Errorf("member directory: %d %s", w.code, w.body)
	}
	if strings.Contains(w.body, `"password"`) {
		return fmt.Errorf("member directory carries a password: %s", w.body)
	}
	return nil
}

func (w *world) driverCatalogMisses(ctx context.Context, driver string) error {
	if err := w.api(http.MethodGet, "/external/v1/config/drivers", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("driver catalog: %d", w.code)
	}
	if strings.Contains(w.body, `"`+driver+`"`) {
		return fmt.Errorf("driver %q is advertised but no module serves it: %s", driver, w.body)
	}
	return nil
}

// changePasswordAndBack exercises the change on the suite's own account and always restores it. Changing a password
// clears the account's session, so every sign-in here installs the token it mints.
func (w *world) changePasswordAndBack(ctx context.Context, next string) error {
	change := func(from, to string) error {
		if err := w.api(http.MethodPost, "/external/v1/auth/password",
			map[string]string{"oldPassword": from, "newPassword": to}, nil); err != nil {
			return err
		}
		if w.code != http.StatusOK && w.code != http.StatusNoContent {
			return fmt.Errorf("change password: %d %s", w.code, w.body)
		}
		return nil
	}
	if err := change(adminPassword, next); err != nil {
		return err
	}
	reverted := false
	defer func() {
		if !reverted {
			_ = change(next, adminPassword)
			_, _ = w.signIn(adminEmail, adminPassword)
		}
	}()
	if _, err := w.signIn(adminEmail, next); err != nil {
		return err
	}
	if err := change(next, adminPassword); err != nil {
		return err
	}
	reverted = true
	_, err := w.signIn(adminEmail, adminPassword)
	return err
}

func (w *world) changePasswordRefused(ctx context.Context) error {
	if err := w.api(http.MethodPost, "/external/v1/auth/password",
		map[string]string{"oldPassword": "not-the-password", "newPassword": "irrelevant-value"}, nil); err != nil {
		return err
	}
	if w.code == http.StatusOK || w.code == http.StatusNoContent {
		return fmt.Errorf("a wrong old password changed the account: %d %s", w.code, w.body)
	}
	return nil
}

// signOutAndBack revokes the session and proves it stopped answering, then signs in again and installs the new token:
// an account holds one session at a time, so the rest of the run needs it back. The deferred sign-in restores it even
// when an assertion fails.
func (w *world) signOutAndBack(ctx context.Context) error {
	revoked, err := w.signIn(adminEmail, adminPassword)
	if err != nil {
		return err
	}
	defer func() { _, _ = w.signIn(adminEmail, adminPassword) }()
	if err := w.api(http.MethodPost, "/external/v1/auth/logout", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK && w.code != http.StatusNoContent {
		return fmt.Errorf("sign out: %d %s", w.code, w.body)
	}
	w.headers["Authorization"] = "Bearer " + revoked
	if err := w.api(http.MethodGet, "/external/v1/me", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusUnauthorized {
		return fmt.Errorf("the signed-out session still answers: %d %s", w.code, w.body)
	}
	return nil
}
