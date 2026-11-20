package constants

// Error codes carried in every failure envelope, next to the status.
const (
	CodeInvalidRequest  = "invalid_request"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeUnsupported     = "unsupported"
	CodeRehearsalFailed = "rehearsal_failed"
	CodeInternal        = "internal"
)
