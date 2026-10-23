package entity

// Global roles. Exactly one account holds Super: the env root, named by
// NOXIS_ADMIN_EMAIL. An Admin runs the instance — directory, bans and every
// connection's grants — but cannot promote, and cannot manage connections.
// Everyone else is a Member whose reach is per-connection grants.
const (
	RoleSuper  = "super"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Account statuses. A banned account keeps its grants but cannot authenticate.
const (
	StatusActive = "active"
	StatusBanned = "banned"
)

// User is a local account or an OIDC identity reference. Subject is the stable
// key: "local:<login>" for a local account, the OIDC sub otherwise. Email is the
// account identifier: a local account signs in with it. Role is the one global
// role. Password is empty for OIDC rows, which exist so grants and the directory
// can reference every authenticated principal.
type User struct {
	Subject     string   `json:"subject" validate:"required"`
	Email       string   `json:"email,omitempty" validate:"omitempty,email"`
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"-"`
	Groups      []string `json:"groups"`
	Role        string   `json:"role" validate:"required,oneof=super admin member"`
	Status      string   `json:"status" validate:"required,oneof=active banned"`
	LastLoginAt int64    `json:"lastLoginAt"`
	CreatedAt   int64    `json:"createdAt"`
}

// LocalSubject namespaces a local account, so it can never collide with an OIDC
// subject.
func LocalSubject(login string) string { return "local:" + login }

// LocalUsername returns the login name for a local subject, or "" when the
// subject is not local.
func LocalUsername(subject string) string {
	const prefix = "local:"
	if len(subject) <= len(prefix) || subject[:len(prefix)] != prefix {
		return ""
	}
	return subject[len(prefix):]
}

// IsSuper reports whether the account holds the single global role.
func (u User) IsSuper() bool { return u.Role == RoleSuper }

// IsAdmin reports whether the account holds a global management role.
func (u User) IsAdmin() bool { return u.Role == RoleSuper || u.Role == RoleAdmin }

// Active reports whether the account may authenticate.
func (u User) Active() bool { return u.Status != StatusBanned }

// Identity renders the account as the request principal.
func (u User) Identity() Identity {
	name := u.Username
	if name == "" {
		name = u.Subject
	}
	return Identity{
		Subject:     u.Subject,
		Name:        name,
		Email:       u.Email,
		Groups:      u.Groups,
		Role:        u.Role,
		LastLoginAt: u.LastLoginAt,
	}
}

// Validate checks the wire fields.
func (u User) Validate() error { return validate.Struct(u) }
