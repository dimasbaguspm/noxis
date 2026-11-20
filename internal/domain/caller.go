package domain

// Caller is the authenticated identity of one request: the token's name and the
// scopes it was declared with. It is the only identity anything downstream sees, and
// its name is what lands in the journal.
type Caller struct {
	Name   string
	Scopes []string
}

// Has reports whether the caller holds scope.
func (c Caller) Has(scope string) bool {
	for _, held := range c.Scopes {
		if held == scope {
			return true
		}
	}
	return false
}

// HasAll reports whether the caller holds every scope in want.
func (c Caller) HasAll(want []string) bool {
	for _, scope := range want {
		if !c.Has(scope) {
			return false
		}
	}
	return true
}

// Anonymous is the caller of a request that carried no usable token.
var Anonymous = Caller{}
