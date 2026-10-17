package kit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStubRecordsThenReplays(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-From", "live")
		_, _ = w.Write([]byte(`{"value":42}`))
	}))
	defer srv.Close()

	rec := &Client{BaseURL: srv.URL, Stub: &Stub{Dir: dir, Record: true}}
	if _, err := rec.Send(Request{Method: http.MethodPost, Path: "/engine/query"}); err != nil {
		t.Fatalf("record: %v", err)
	}

	// Offline: no server in reach, the stored response is served instead.
	off := &Client{BaseURL: "http://127.0.0.1:1", Stub: &Stub{Dir: dir}}
	resp, err := off.Send(Request{Method: http.MethodPost, Path: "/engine/query"})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if resp.Status != http.StatusOK || resp.String() != `{"value":42}` {
		t.Errorf("replayed %d %q", resp.Status, resp.String())
	}
	if resp.Headers.Get("X-From") != "live" {
		t.Errorf("headers not replayed: %v", resp.Headers)
	}
}

func TestStubMissingIsActionable(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1", Stub: &Stub{Dir: t.TempDir()}}
	_, err := c.Send(Request{Method: http.MethodGet, Path: "/nope"})
	if err == nil || !strings.Contains(err.Error(), "UAT_RECORD=1") {
		t.Errorf("missing stub error = %v, want a UAT_RECORD hint", err)
	}
}
