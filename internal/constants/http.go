package constants

// Wire names: headers, media types and the paths that never move.
const (
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-Id"
	HeaderContentType   = "Content-Type"

	MediaTypeJSON = "application/json; charset=utf-8"
	MediaTypeYAML = "application/yaml"

	SchemeBearer = "Bearer"

	PathHealthz     = "/healthz"
	PathOpenAPIJSON = "/openapi.json"
	PathOpenAPIYAML = "/openapi.yaml"
	PathPrefix      = "/v1"
	PathWhoami      = "/v1/whoami"
	PathTargets     = "/v1/targets"
	PathProposals   = "/v1/proposals"
)

// The server's deployment contract. These are constants on purpose: a state file that can be moved
// by an environment variable is a journal that can be lost by a typo. The binary takes -addr and
// -state flags for development and for the acceptance suite; the environment does not.
const (
	ListenAddr = ":9000"
	StatePath  = "/data/noxis.db"
)

// ConsoleOrigins are the origins allowed to call the API from a browser. The console is standalone;
// while it is a development server this is the one origin there is.
var ConsoleOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
