package client

// The client's copy of the API's paths. The server writes the same strings in its route table, and
// nothing checks them against each other at compile time: the acceptance suite boots the real
// binary and calls it, which is the only proof that matters. That is cheaper than a constant two
// packages share, which would make every path change a change in a third place.
const (
	pathHealthz   = "/healthz"
	pathWhoami    = "/v1/whoami"
	pathTargets   = "/v1/targets"
	pathOpenAPI   = "/openapi.json"
	pathProposals = "/v1/proposals"
)
