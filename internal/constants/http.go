package constants

// Wire names: headers, media types and the paths that never move.
const (
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-Id"
	HeaderContentType   = "Content-Type"

	MediaTypeJSON = "application/json; charset=utf-8"

	SchemeBearer = "Bearer"

	PathHealthz   = "/healthz"
	PathPrefix    = "/v1"
	PathWhoami    = "/v1/whoami"
	PathTargets   = "/v1/targets"
	PathProposals = "/v1/proposals"
)
