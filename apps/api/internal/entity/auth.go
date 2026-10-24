package entity

// Identity is the authenticated caller, derived from the bearer token on every request. A blended OIDC login carries the local account's subject and role, so grants apply unchanged.
type Identity struct {
	Subject     string   `json:"sub"`
	Name        string   `json:"name,omitempty"`
	Email       string   `json:"email,omitempty"`
	Groups      []string `json:"groups"`
	Role        string   `json:"role"`
	LastLoginAt int64    `json:"lastLoginAt,omitempty"`
	// EmailVerified is the issuer's claim, never the store's truth; only a verified email may blend an OIDC login into a local account.
	EmailVerified bool `json:"-"`
}

// IsSuper reports whether the caller holds the env root's single global role.
func (i Identity) IsSuper() bool { return i.Role == RoleSuper }

// IsAdmin reports whether the caller may run the instance: the super admin or a promoted admin.
func (i Identity) IsAdmin() bool { return i.Role == RoleSuper || i.Role == RoleAdmin }
