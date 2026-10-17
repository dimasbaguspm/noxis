package entity

// Access levels a member can hold on one connection, weakest to strongest.
// Enforcement is "at least this level", so read < write < admin.
const (
	LevelRead  = "read"
	LevelWrite = "write"
	LevelAdmin = "admin"

	// Member kinds a permission can target.
	MemberUser  = "user"
	MemberGroup = "group"
)

var levelRank = map[string]int{
	LevelRead:  1,
	LevelWrite: 2,
	LevelAdmin: 3,
}

// LevelAllows reports whether have is at least need. An unknown level never
// allows (default deny).
func LevelAllows(have, need string) bool {
	haveRank, ok := levelRank[have]
	if !ok {
		return false
	}
	needRank, ok := levelRank[need]
	return ok && haveRank >= needRank
}

// StrongestLevel returns the strongest known level in levels, or "" when none
// is known. It collapses a caller's matching permissions into one effective level.
func StrongestLevel(levels ...string) string {
	best := ""
	for _, level := range levels {
		rank, ok := levelRank[level]
		if !ok {
			continue
		}
		if best == "" || rank > levelRank[best] {
			best = level
		}
	}
	return best
}

// Permission binds one member to one connection at a level. A member is either
// a user subject (local:<name> or an OIDC sub) or a group name. Noxis stores the
// policy only — identity and membership come from the account row or the issuer.
type Permission struct {
	Connection string `json:"connection,omitempty"`
	Member     string `json:"member" validate:"required"`
	MemberType string `json:"memberType" validate:"required,oneof=user group"`
	Level      string `json:"level" validate:"required,oneof=read write admin"`
	ExpiresAt  int64  `json:"expiresAt"`
	CreatedBy  string `json:"createdBy,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}

// Expired reports whether the grant has lapsed at now. A zero expiry means
// "no expiry" and is only expected on rows written before the rule existed.
func (p Permission) Expired(now int64) bool {
	return p.ExpiresAt > 0 && p.ExpiresAt <= now
}

// Validate checks the member, member type and level are present and known.
func (p Permission) Validate() error {
	return validate.Struct(p)
}
