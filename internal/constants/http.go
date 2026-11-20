package constants

// Wire names: headers, media types and the paths that never move.
const (
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-Id"
	HeaderContentType   = "Content-Type"

	MediaTypeJSON = "application/json; charset=utf-8"
	MediaTypeYAML = "application/yaml"

	SchemeBearer = "Bearer"
)

// The server's deployment contract. These are constants on purpose: a state file that can be moved
// by an environment variable is a journal that can be lost by a typo. The binary takes -addr and
// -state flags for development and for the acceptance suite; the environment does not.
const (
	ListenAddr = ":9000"
	// ConsolePort is where `noxis web serve` listens. A constant, like the API's address: a console
	// that moves by environment is a bookmark that stops working.
	ConsolePort = 3000
	StatePath   = "/data/noxis.db"
)

// ConsoleOrigins are the origins allowed to call the API from a browser. A variable until #807 makes
// the deployment say which console it has.
var ConsoleOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
