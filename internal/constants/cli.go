package constants

import "time"

// DefaultAPIURL is where a noxisd on this machine listens: the same port the server binds.
const DefaultAPIURL = "http://localhost:9000"

// AuthCheckWindow is how long a stored token is trusted without asking the server again. Long
// enough that a script of twenty commands asks once, short enough that a revoked token is noticed
// before the work it was about to do.
const AuthCheckWindow = 60 * time.Second
