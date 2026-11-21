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
	// ConsolePort is where `noxis web serve` listens. A constant, like the API's address: a console
	// that moves by environment is a bookmark that stops working.
	ConsolePort = 3000
	// ConsoleListen is the interface `noxis web serve` binds. A console carries the operator's token, so
	// the default is the loopback: a laptop at a conference does not hand it to the coffee shop. A
	// deployment behind a proxy names the interface it wants with --listen.
	ConsoleListen = "127.0.0.1"
	StatePath     = "/data/noxis.db"
)

// DefaultConsoleOrigins are the browser origins a development console is served from: the Vite dev
// server and the same thing addressed by address. A deployment names its own in CONSOLE_ORIGINS.
const DefaultConsoleOrigins = "http://localhost:5173,http://127.0.0.1:5173"
