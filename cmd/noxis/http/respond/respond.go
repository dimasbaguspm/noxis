// Package respond is the one place that knows the shape of an API answer, so a refused
// request and a failed handler look the same to a client.
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
)

// Envelope is every answer this API gives, on a success and on a refusal alike. It is exported
// because it is the document's shape as much as the client's: one type, one place a field can be
// added, and the generated spec names it.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Meta is what a list carries beside its page.
type Meta struct {
	NextCursor string `json:"next_cursor,omitempty"`
}

// Error is a refusal a client can act on: a stable code and a sentence written for the person
// reading it.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OK writes one resource.
func OK(w http.ResponseWriter, data any) { write(w, http.StatusOK, Envelope{Data: data}) }

// Created writes the resource a POST minted, at the status that says so.
func Created(w http.ResponseWriter, data any) { write(w, http.StatusCreated, Envelope{Data: data}) }

// Accepted writes the work a POST started but did not finish, at the status that says "poll me".
func Accepted(w http.ResponseWriter, data any) { write(w, http.StatusAccepted, Envelope{Data: data}) }

// Page writes a list and names the cursor that follows it, if there is one.
func Page(w http.ResponseWriter, data any, next string) {
	write(w, http.StatusOK, Envelope{Data: data, Meta: &Meta{NextCursor: next}})
}

// Fail maps a domain sentinel onto a status and a code. An error it does not recognise
// is logged whole and answered generically, so no internal message reaches a client.
func Fail(w http.ResponseWriter, log *slog.Logger, err error) {
	status, code := http.StatusInternalServerError, constants.CodeInternal
	switch {
	case errors.Is(err, domain.ErrInvalid):
		status, code = http.StatusBadRequest, constants.CodeInvalidRequest
	case errors.Is(err, domain.ErrUnauthorized):
		status, code = http.StatusUnauthorized, constants.CodeUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		status, code = http.StatusForbidden, constants.CodeForbidden
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, constants.CodeNotFound
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusConflict, constants.CodeConflict
	case errors.Is(err, domain.ErrUnsupported):
		status, code = http.StatusUnprocessableEntity, constants.CodeUnsupported
	}
	if code == constants.CodeInternal {
		log.Error("request failed", "error", err)
		Refuse(w, status, code, "the server could not complete the request")
		return
	}
	Refuse(w, status, code, cause(err))
}

// cause drops the sentinel's own name from a wrapped error, so a client reads
// "Error 1045 ..." instead of "unsupported: Error 1045 ..." under a code that already says it.
func cause(err error) string {
	text := err.Error()
	for _, name := range []string{"invalid", "not found", "forbidden", "unauthorized", "conflict", "unsupported"} {
		if rest, found := strings.CutPrefix(text, name+": "); found {
			return rest
		}
	}
	return text
}

// Refuse writes a failure without an error value: the middlewares refuse before any use
// case exists to return one.
func Refuse(w http.ResponseWriter, status int, code, message string) {
	write(w, status, Envelope{Error: &Error{Code: code, Message: message}})
}

func write(w http.ResponseWriter, status int, body Envelope) {
	buf, err := json.Marshal(body)
	if err != nil {
		// The envelope is ours and always marshals; this branch exists so a future
		// non-serialisable payload cannot answer 200 with an empty body.
		Refuse(w, http.StatusInternalServerError, constants.CodeInternal, "the server could not write the response")
		return
	}
	w.Header().Set(constants.HeaderContentType, constants.MediaTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}
