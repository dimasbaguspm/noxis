package sql

import (
	"strings"

	"noxis/internal/constants"
)

// The three predicates below are the only driver-name questions left outside dialect.go, and only the change verifier still asks them: it runs two different proofs (rolled-back transaction, scratch schema) whose steps differ per engine. Splitting that file by strategy retires them.

// isSQLite reports whether the driver is the file-backed SQL database.
func isSQLite(driver string) bool {
	return strings.EqualFold(driver, constants.DriverSQLite)
}

// isPostgres reports whether the driver speaks the postgres protocol.
func isPostgres(driver string) bool {
	switch strings.ToLower(driver) {
	case constants.DriverPostgres, "postgresql":
		return true
	}
	return false
}

// isMySQL reports whether the driver speaks the mysql protocol.
func isMySQL(driver string) bool {
	switch strings.ToLower(driver) {
	case constants.DriverMySQL, constants.DriverMariaDB:
		return true
	}
	return false
}
