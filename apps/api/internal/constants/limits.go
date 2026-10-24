package constants

// Limits bound the operator surfaces: how much a single request may pull, so no caller can force the whole store into one response.
const (
	// DefaultPageSize and MaxPageSize bound list endpoints.
	DefaultPageSize = 20
	MaxPageSize     = 100

	// ScanLimit bounds a memory-cache key browse, and MaxScanLimit the caller's ask.
	ScanLimit    = 500
	MaxScanLimit = 5000
)
