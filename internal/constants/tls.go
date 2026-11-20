package constants

import "slices"

// Transport security for a target's connection. The three names map one to one onto the
// driver's own options, so nothing is implied that is not true: `skip-verify` encrypts
// without proving the certificate, `verify` does both.
const (
	TLSPreferred  = "preferred"
	TLSSkipVerify = "skip-verify"
	TLSVerify     = "verify"
)

// TLSModes is every mode a target may declare.
var TLSModes = []string{TLSPreferred, TLSSkipVerify, TLSVerify}

// IsTLSMode reports whether s is a mode this build speaks.
func IsTLSMode(s string) bool {
	return slices.Contains(TLSModes, s)
}

// DefaultMySQLPort is where both engines listen unless told otherwise.
const DefaultMySQLPort = 3306
