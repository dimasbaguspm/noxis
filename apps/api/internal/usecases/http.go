package usecases

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/auth"
)

// ErrUnauthorized maps to 401: the caller is missing though the operation
// needs one (a local-account action reachable only when authenticated).
var ErrUnauthorized = errors.New("invalid or missing api key")

// errLocalAccountRequired reports a local-only action asked of an OIDC caller.
var errLocalAccountRequired = errors.New("a local account is required for this action")

// errKeyRequired reports the value route called without a key.
var errKeyRequired = errors.New("key is required")

// errFlushConfirm reports a flush whose confirmation did not repeat the name.
func errFlushConfirm(name string) error {
	return fmt.Errorf("flushing %s needs its name typed as confirmation", name)
}

// respond writes v as a JSON 200. A nil v writes an empty 200 body.
func respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// respondNoContent writes 204 for operations with an empty response.
func respondNoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// decode reads the request body as JSON into v.
func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

// fail maps err to a status and writes the error envelope. It is the single
// place the API's errors become HTTP: a missing row and a missing engine both
// read as "not here", bad credentials and a missing caller as 401.
func fail(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	switch {
	case errors.Is(err, interfaces.ErrNotFound), errors.Is(err, interfaces.ErrUnsupported):
		code = http.StatusNotFound
	case errors.Is(err, ErrUnauthorized), errors.Is(err, auth.ErrInvalidCredentials):
		code = http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		code = http.StatusForbidden
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: err.Error()})
}
