package entity

// ServerInfo is what a connected server says about itself. Version is detected from the server, never typed by an operator: a stored version would go stale and silently route a change into the wrong proof.
type ServerInfo struct {
	// Engine is the driver the connection speaks (postgres, mysql, sqlite).
	Engine string `json:"engine"`
	// Version is the version string the server reported.
	Version string `json:"version"`
	// Capabilities names what this profile lets the console prove, in the operator's words.
	Capabilities []string `json:"capabilities"`
}
