// Package constants holds the values every layer names instead of repeating:
// scopes, drivers, wire names, limits, error codes.
package constants

import "slices"

// The five scopes a token can hold. read and propose belong to an agent; approve and
// apply belong to a human; admin overrides separation of duties.
const (
	ScopeRead    = "read"
	ScopePropose = "propose"
	ScopeApprove = "approve"
	ScopeApply   = "apply"
	ScopeAdmin   = "admin"
)

// Scopes is every scope a token or a route may name.
var Scopes = []string{ScopeRead, ScopePropose, ScopeApprove, ScopeApply, ScopeAdmin}

// IsScope reports whether s is a scope this build knows.
func IsScope(s string) bool {
	return slices.Contains(Scopes, s)
}
