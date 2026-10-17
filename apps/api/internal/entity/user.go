package entity

// User is a local account or an OIDC identity reference. Subject is the stable
// key: "local:<username>" for a local account, the OIDC sub otherwise. Username
// and Password are empty for OIDC rows, which exist so grants and the user
// directory can reference every authenticated principal.
type User struct {
	Subject     string   `json:"subject" validate:"required"`
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"-"`
	Groups      []string `json:"groups"`
	Super       bool     `json:"super"`
	Disabled    bool     `json:"disabled"`
	LastLoginAt int64    `json:"lastLoginAt"`
}

// LocalSubject namespaces a local account, so it can never collide with an OIDC
// subject.
func LocalSubject(username string) string { return "local:" + username }

// LocalUsername returns the account name for a local subject, or "" when the
// subject is not local.
func LocalUsername(subject string) string {
	const prefix = "local:"
	if len(subject) <= len(prefix) || subject[:len(prefix)] != prefix {
		return ""
	}
	return subject[len(prefix):]
}

// Identity renders the account as the request principal.
func (u User) Identity() Identity {
	name := u.Username
	if name == "" {
		name = u.Subject
	}
	return Identity{Subject: u.Subject, Name: name, Groups: u.Groups, Super: u.Super}
}

// Validate checks the wire fields.
func (u User) Validate() error { return validate.Struct(u) }
