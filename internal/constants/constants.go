// Package constants centralises the app's universal defined values: version,
// group naming, status states. Module-local domain strings stay with their
// module (doorman/constants.go); user-facing error messages live next to the
// logic that raises them.
package constants

const (
	// Version is the app version (user-agent, reports).
	Version = "0.1.0"

	// ConsumerGroupPrefix names each slave's durable consumer group
	// (ConsumerGroupPrefix + node id). Group names are stable and distinct
	// from the env-configurable topic prefix.
	ConsumerGroupPrefix = "noxis."

	// Status states crossing the wire (slave heartbeats, master merge).
	StateOK      = "ok"
	StateError   = "error"
	StateUp      = "up"
	StateOffline = "offline"
)
