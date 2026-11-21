package constants

// Wire names: headers, media types and the paths that never move.
const (
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-Id"
	HeaderContentType   = "Content-Type"
	// HeaderIdempotencyKey names a create so a retry answers the row the first call made.
	HeaderIdempotencyKey = "Idempotency-Key"

	MediaTypeJSON = "application/json; charset=utf-8"
	MediaTypeYAML = "application/yaml"

	SchemeBearer = "Bearer"
)

// The server's deployment contract. These are constants on purpose: a state file that can be moved
// by an environment variable is a journal that can be lost by a typo. The binary takes -addr and
// -state flags for development and for the acceptance suite; the environment does not.
const (
	ListenAddr = ":9000"
	StatePath  = "/data/noxis.db"
)
