package constants

// Error codes carried in every failure envelope, next to the status.
const (
	CodeInvalidRequest = "invalid_request"
	CodeUnauthorized   = "unauthorized"
	CodeForbidden      = "forbidden"
	CodeNotFound       = "not_found"
	CodeConflict       = "conflict"
	CodeUnsupported    = "unsupported"
	// CodeClient marks a failure that never reached the server: a flag, a file, a session that was never
	// signed in. The API's codes name a refusal; this one names the command that could not ask.
	CodeClient          = "client"
	CodeRehearsalFailed = "rehearsal_failed"
	CodeInternal        = "internal"
)
