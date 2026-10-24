package uat

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	"noxis/test/uat/kit"
)

// httpSteps are the generic request controls: headers, the default bearer, and cookies set per test, plus a raw-payload sender.
func (w *world) httpSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I set header "([^"]*)" to "([^"]*)"$`, w.setHeader)
	sc.Step(`^I set the auth token to "([^"]*)"$`, w.setAuthToken)
	sc.Step("I clear the auth header", w.clearAuthHeader)
	sc.Step(`^I set cookie "([^"]*)" to "([^"]*)"$`, w.setCookie)
	sc.Step(`^the response header "([^"]*)" is "([^"]*)"$`, w.responseHeaderIs)
	sc.Step(`^the response status is (\d+)$`, w.responseStatusIs)
	sc.Step(`^the response matches snapshot "([^"]*)"$`, w.responseMatchesSnapshot)
	sc.Step(`^I send a (\w+) to "([^"]*)" with payload:$`, w.sendPayload)
}

func (w *world) setHeader(ctx context.Context, name, value string) error {
	w.headers[name] = value
	return nil
}

func (w *world) setAuthToken(ctx context.Context, tok string) error {
	w.headers["Authorization"] = "Bearer " + tok
	return nil
}

// clearAuthHeader makes the default bearer absent for this test: the kit removes a header whose override value is empty.
func (w *world) clearAuthHeader(ctx context.Context) error {
	w.headers["Authorization"] = ""
	return nil
}

func (w *world) setCookie(ctx context.Context, name, value string) error {
	w.cookies = append(w.cookies, &http.Cookie{Name: name, Value: value})
	return nil
}

func (w *world) responseHeaderIs(ctx context.Context, name, want string) error {
	if got := w.resp.Headers.Get(name); got != want {
		return fmt.Errorf("response header %s = %q, want %q", name, got, want)
	}
	return nil
}

func (w *world) responseStatusIs(ctx context.Context, want int) error {
	if w.resp.Status != want {
		return fmt.Errorf("response status = %d, want %d", w.resp.Status, want)
	}
	return nil
}

// responseMatchesSnapshot compares the last response against a committed golden under test/uat/snapshots.
func (w *world) responseMatchesSnapshot(ctx context.Context, name string) error {
	return w.snapshots.Match(name, w.resp.Body)
}

// sendPayload sends a request whose body is the step's docstring, verbatim, and decodes the reply into the shared result grid.
func (w *world) sendPayload(ctx context.Context, method, path string, payload *godog.DocString) error {
	if err := w.send(kit.Request{Method: method, Path: path, Body: []byte(payload.Content)}); err != nil {
		return err
	}
	w.result = resultBody{}
	return w.resp.Decode(&w.result)
}
