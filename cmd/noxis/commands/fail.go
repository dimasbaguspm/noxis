package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"noxis/internal/client"
	"noxis/internal/constants"
)

// Failure is the machine shape of a failure: the code the API answered with, the message a human
// reads, and the status when the server answered at all. An agent branches on `code`, which is the
// same vocabulary the wire uses, so it never has to match a sentence to learn what went wrong.
type Failure struct {
	Status  int    `json:"status,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Report writes one failure to w: as the machine shape when the caller asked for it, and as the one
// line a person reads otherwise. The error a command returned is unchanged either way — this is the
// last hundred milliseconds of the process, not a second error channel.
func Report(w io.Writer, err error, machine bool) {
	if machine {
		buf, marshalErr := json.MarshalIndent(struct {
			Error Failure `json:"error"`
		}{Error: failureOf(err)}, "", "  ")
		if marshalErr == nil {
			// The write is the whole job; a broken pipe on stderr leaves nothing to report to.
			_, _ = fmt.Fprintln(w, string(buf))
			return
		}
	}
	_, _ = fmt.Fprintln(w, "error:", err)
}

// failureOf maps a returned error onto the wire's vocabulary. One that reached the API already
// carries a code; one that did not — a flag, a file, a session that was never signed in — is this
// client's own refusal and says so, with no status to name.
func failureOf(err error) Failure {
	var refused *client.Error
	if !errors.As(err, &refused) {
		return Failure{Code: constants.CodeClient, Message: err.Error()}
	}
	code := refused.Code
	if code == "" {
		// A failure envelope without a code is a server that answered off-contract.
		code = constants.CodeInternal
	}
	message := refused.Message
	if message == "" {
		message = http.StatusText(refused.Status)
	}
	return Failure{Status: refused.Status, Code: code, Message: message}
}
