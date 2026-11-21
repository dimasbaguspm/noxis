package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// The placeholder only exists before a build, so a built console cannot exercise it. The branch still
// has to work: it is what an operator sees when they built an image without running `make web`. The page
// is served as it was committed — there is nothing to render into it, because the API is this origin.
func TestAPlaceholderIsServedAsItWasCommitted(t *testing.T) {
	const page = `<div id="root"></div><p>not built</p>`
	dist := fstest.MapFS{
		"index.html": {Data: []byte(page)},
		"NOT_BUILT":  {Data: []byte("")},
	}
	console, err := newFrom(dist)
	if err != nil {
		t.Fatalf("newFrom: %v", err)
	}
	if console.Built() {
		t.Fatal("a placeholder was reported as a build")
	}

	rec := httptest.NewRecorder()
	console.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != page {
		t.Fatalf("the placeholder was altered on the way out:\n%s", rec.Body.String())
	}
}

// A dist without an index is not a console, and the caller has to hear about it here rather than in a
// browser.
func TestAConsoleWithoutIndexIsAnError(t *testing.T) {
	if _, err := newFrom(fstest.MapFS{}); err == nil {
		t.Fatal("an empty dist built a console")
	}
}
