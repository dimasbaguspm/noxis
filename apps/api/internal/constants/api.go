package constants

// Version is the build version reported by the CLI and health surfaces.
const Version = "0.1.0"

// APIPrefix is the versioned mount point of the public API.
const APIPrefix = "/external/v1"

// Console paths: the API serves the embedded SPA on every path that is not under APIPrefix.
const (
	// ConsoleIndex is the built SPA shell; ConsolePlaceholder stands in when the console was never built.
	ConsoleIndex       = "index.html"
	ConsolePlaceholder = "placeholder.html"
	// ConsoleAssets is the directory of content-hashed bundles, the only files safe to cache forever.
	ConsoleAssets = "assets"
)
