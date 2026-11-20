package respond_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
	"testing"
)

// quiet is the logger Fail takes. The test asserts the body; the log line is the operator's copy.
func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(&strings.Builder{}, nil)) }

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var body envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not the envelope: %v (%s)", err, rec.Body.String())
	}
	return body
}

// Every sentinel gets one status and one code, and the wrapper that carried it does not leak into
// the message: the code already says what the class of the failure is.
func TestFailMapsEverySentinel(t *testing.T) {
	for _, tc := range []struct {
		err     error
		status  int
		code    string
		message string
	}{
		{
			fmt.Errorf("%w: name is required", domain.ErrInvalid), http.StatusBadRequest,
			constants.CodeInvalidRequest, "name is required",
		},
		{
			fmt.Errorf("%w: no token", domain.ErrUnauthorized), http.StatusUnauthorized,
			constants.CodeUnauthorized, "no token",
		},
		{
			fmt.Errorf("%w: scope missing", domain.ErrForbidden), http.StatusForbidden,
			constants.CodeForbidden, "scope missing",
		},
		{
			fmt.Errorf("%w: proposal abc", domain.ErrNotFound), http.StatusNotFound,
			constants.CodeNotFound, "proposal abc",
		},
		{
			fmt.Errorf("%w: proposal is applied", domain.ErrConflict), http.StatusConflict,
			constants.CodeConflict, "proposal is applied",
		},
		{
			fmt.Errorf("%w: the engine cannot do that", domain.ErrUnsupported), http.StatusUnprocessableEntity,
			constants.CodeUnsupported, "the engine cannot do that",
		},
	} {
		rec := httptest.NewRecorder()
		respond.Fail(rec, quiet(), tc.err)

		if rec.Code != tc.status {
			t.Fatalf("%v answered %d, want %d", tc.err, rec.Code, tc.status)
		}
		body := decode(t, rec)
		if body.Error == nil {
			t.Fatalf("%v: no error in the envelope", tc.err)
		}
		if body.Error.Code != tc.code {
			t.Fatalf("%v: code %q, want %q", tc.err, body.Error.Code, tc.code)
		}
		if body.Error.Message != tc.message {
			t.Fatalf("%v: message %q, want %q", tc.err, body.Error.Message, tc.message)
		}
	}
}

// An error nobody classified is a bug in this build, not a message for the client: the cause is
// logged and the body says nothing about it.
func TestFailHidesAnUnknownCause(t *testing.T) {
	var logged strings.Builder
	log := slog.New(slog.NewTextHandler(&logged, nil))

	rec := httptest.NewRecorder()
	respond.Fail(rec, log, errors.New("pq: password authentication failed for user noxis"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := decode(t, rec)
	if body.Error == nil || body.Error.Code != constants.CodeInternal {
		t.Fatalf("body = %+v, want the internal code", body)
	}
	if strings.Contains(body.Error.Message, "password") {
		t.Fatalf("message = %q, it leaked the cause", body.Error.Message)
	}
	if !strings.Contains(logged.String(), "password authentication failed") {
		t.Fatal("the cause was not logged: an operator has nothing to read")
	}
}

// A wrapped sentinel is still recognised: errors.Is is the contract, not string equality.
func TestFailFollowsTheWrapChain(t *testing.T) {
	inner := fmt.Errorf("%w: forbidden", domain.ErrForbidden)
	outer := fmt.Errorf("deciding proposal: %w", inner)

	rec := httptest.NewRecorder()
	respond.Fail(rec, quiet(), outer)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 through two wraps", rec.Code)
	}
}

// The success shapes are what a client parses, so their statuses are pinned too.
func TestSuccessStatuses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		write  func(http.ResponseWriter)
	}{
		{"ok", http.StatusOK, func(w http.ResponseWriter) { respond.OK(w, map[string]string{"a": "b"}) }},
		{"created", http.StatusCreated, func(w http.ResponseWriter) { respond.Created(w, map[string]string{"a": "b"}) }},
		{"accepted", http.StatusAccepted, func(w http.ResponseWriter) { respond.Accepted(w, map[string]string{"a": "b"}) }},
		{"page", http.StatusOK, func(w http.ResponseWriter) { respond.Page(w, []string{}, "next") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.write(rec)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if got := rec.Header().Get(constants.HeaderContentType); got != constants.MediaTypeJSON {
				t.Fatalf("content type = %q", got)
			}
		})
	}
}

// Refuse is the middlewares' path: no error value exists yet, and the message is written as given.
func TestRefuseWritesWhatItIsGiven(t *testing.T) {
	rec := httptest.NewRecorder()
	respond.Refuse(rec, http.StatusUnauthorized, constants.CodeUnauthorized, "no token")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	body := decode(t, rec)
	if body.Error == nil || body.Error.Message != "no token" {
		t.Fatalf("body = %+v", body)
	}
}
