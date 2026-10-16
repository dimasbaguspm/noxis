package entity

// ErrorResponse is the machine-readable payload every failing route returns;
// `error` holds the human-readable reason.
type ErrorResponse struct {
	Error string `json:"error"`
}

// StatusResponse is the success envelope for operations with no payload
// beyond an outcome word (ok, flushed).
type StatusResponse struct {
	Status string `json:"status"`
}
