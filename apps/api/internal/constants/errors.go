package constants

import "errors"

// Cross-cutting sentinel errors. They live in the lowest layer so every layer
// can raise and match them without importing a sibling; errors that describe a
// single module (sql.ErrUnverifiable) stay with the module that raises them.
var (
	// ErrNotFound reports a missing row. The API maps it to 404.
	ErrNotFound = errors.New("not found")
	// ErrUnsupported reports an engine name with no module, or an engine that
	// does not serve the requested operation. The API maps it to 404.
	ErrUnsupported = errors.New("unsupported engine")
	// ErrForbidden reports an authorization failure. The API maps it to 403.
	ErrForbidden = errors.New("forbidden: global admin required")
	// ErrUnauthorized reports a missing caller. The API maps it to 401.
	ErrUnauthorized = errors.New("invalid or missing api key")
	// ErrInvalidCredentials reports a failed local login. The API maps it to 401.
	ErrInvalidCredentials = errors.New("invalid credentials")
)
