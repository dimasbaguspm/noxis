package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestActionReplaysBodyToTheChosenHandler guards the dispatcher: the target use
// case decodes the same payload the action was read from.
func TestActionReplaysBodyToTheChosenHandler(t *testing.T) {
	var got string
	h := actionHandler(map[string]http.Handler{
		"approve": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			got = string(body)
			w.WriteHeader(http.StatusOK)
		}),
	})
	payload := `{"action":"approve","note":"ok"}`
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/action", strings.NewReader(payload)))
	if rec.Code != http.StatusOK || got != payload {
		t.Fatalf("status=%d body=%q", rec.Code, got)
	}
}

func TestActionRefusesUnknownAction(t *testing.T) {
	h := actionHandler(map[string]http.Handler{})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/action", strings.NewReader(`{"action":"delete"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
