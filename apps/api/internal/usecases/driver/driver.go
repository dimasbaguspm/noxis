// Package driver holds the driver-catalog use case.
package driver

import (
	"net/http"

	"noxis/internal/constants"
	"noxis/internal/usecases"
)

// DriverInfo is one entry of the driver catalog the console reads to build its
// pickers. A driver is code, not data, so the catalog is hardcoded Go-side and
// this endpoint is the only source: no UI constant drifts from the server.
type DriverInfo struct {
	Driver string `json:"driver"`
	Family string `json:"family"`
	Label  string `json:"label"`
}

var driverCatalog = []DriverInfo{
	{constants.DriverPostgres, constants.FamilySQL, "PostgreSQL"},
	{constants.DriverMySQL, constants.FamilySQL, "MySQL / MariaDB"},
	{constants.DriverSQLite, constants.FamilySQL, "SQLite"},
	{constants.DriverRedis, constants.FamilyMemcache, "Redis"},
	{constants.DriverValkey, constants.FamilyMemcache, "Valkey"},
	{constants.DriverMongo, constants.FamilyMongo, "MongoDB"},
}

// Drivers serves the driver catalog.
type Drivers struct{ usecases.RegistryDeps }

// @Summary List the drivers the console can register
// @Success 200 {array} driver.DriverInfo
// @Router /external/v1/config/drivers [get]
func (uc Drivers) Serve(w http.ResponseWriter, _ *http.Request) { uc.Respond(w, driverCatalog) }
