package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// Respond writes v as a JSON 200. A nil v writes an empty 200 body.
func (d RegistryDeps) Respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// RespondNoContent writes 204 for operations with an empty response.
func (d RegistryDeps) RespondNoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Decode reads the request body as JSON into v.
func (d RegistryDeps) Decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

// Fail maps err to a status, records it on the operation's span and writes the error envelope.
func (d RegistryDeps) Fail(ctx context.Context, w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	switch {
	case errors.Is(err, constants.ErrNotFound), errors.Is(err, constants.ErrUnsupported):
		code = http.StatusNotFound
	case errors.Is(err, constants.ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, constants.ErrUnauthorized), errors.Is(err, constants.ErrInvalidCredentials):
		code = http.StatusUnauthorized
	case errors.Is(err, constants.ErrForbidden):
		code = http.StatusForbidden
	}
	// The use case is the caller of Fail; the status is what the client saw.
	d.Metrics.UsecaseError(ctx, logging.CallerName(1), strconv.Itoa(code))
	logging.ErrorWithCtx(ctx, "request failed", err, "status", code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: err.Error()})
}
