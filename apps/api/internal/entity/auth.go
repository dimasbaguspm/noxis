package entity

// Identity is the authenticated caller, derived from the bearer token on every
// request. Super is the console's one global role: exactly one account (the env
// root) holds it and it never expires. Everything else is per-connection.
type Identity struct {
	Subject     string   `json:"sub"`
	Name        string   `json:"name,omitempty"`
	Email       string   `json:"email,omitempty"`
	Groups      []string `json:"groups"`
	Super       bool     `json:"super"`
	LastLoginAt int64    `json:"lastLoginAt,omitempty"`
}
