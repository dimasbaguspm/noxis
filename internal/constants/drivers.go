package constants

import "slices"

// The two engines this build speaks, both through one module.
const (
	DriverMySQL   = "mysql"
	DriverMariaDB = "mariadb"
)

// Drivers is every driver name a target may declare.
var Drivers = []string{DriverMySQL, DriverMariaDB}

// IsDriver reports whether s names an engine this build speaks.
func IsDriver(s string) bool {
	return slices.Contains(Drivers, s)
}
