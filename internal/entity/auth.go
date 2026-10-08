package entity

// Principal is the authenticated caller, derived from the OIDC token on every
// request. Noxis stores no users: identity and group membership come from the
// issuer, and the groups drive the per-connection grants.
type Principal struct {
	Subject string   `json:"sub"`
	Groups  []string `json:"groups"`
	// Admin reports membership of the configured global-admin group.
	Admin bool `json:"admin"`
}
