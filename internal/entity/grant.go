package entity

// Access levels a group can hold on one connection, weakest to strongest.
// Enforcement is "at least this level", so read < write < admin.
const (
	LevelRead  = "read"
	LevelWrite = "write"
	LevelAdmin = "admin"
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

// Grant binds one OIDC group to one connection at a level. Noxis stores the
// policy only — identity and membership live in the issuer.
type Grant struct {
	Connection string `json:"connection,omitempty"`
	Group      string `json:"group" validate:"required"`
	Level      string `json:"level" validate:"required,oneof=read write admin"`
}

// Validate checks the group and level are present and the level is known.
func (g Grant) Validate() error {
	return validate.Struct(g)
}
