package constants

// Ceilings. Every one of them is a limit the API reports rather than silently applies.
const (
	// DefaultPageLimit and MaxPageLimit bound every cursor-paged list.
	DefaultPageLimit = 50
	MaxPageLimit     = 200

	// DefaultRowsLimit and MaxRowsLimit bound a raw result set, which reports
	// `truncated` instead of paging.
	DefaultRowsLimit = 200
	MaxRowsLimit     = 1000

	// MinTokenSecretLen is the shortest accepted token secret.
	MinTokenSecretLen = 32
	// MaxPages bounds a walk through every page: a cursor that points at itself would otherwise loop
	// forever, and a command that never returns is worse than one that says it stopped.
	MaxPages = 1000
)
