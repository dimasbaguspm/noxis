// Package driver holds the driver-catalog use case.
package driver

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// driverCatalog is the only source of the console's driver pickers: a driver is
// code, not data, so no UI constant can drift from the server.
var driverCatalog = []entity.DriverInfo{
	{Driver: constants.DriverPostgres, Family: constants.FamilySQL, Label: "PostgreSQL"},
	{Driver: constants.DriverMySQL, Family: constants.FamilySQL, Label: "MySQL / MariaDB"},
	{Driver: constants.DriverSQLite, Family: constants.FamilySQL, Label: "SQLite"},
	{Driver: constants.DriverRedis, Family: constants.FamilyMemcache, Label: "Redis"},
	{Driver: constants.DriverValkey, Family: constants.FamilyMemcache, Label: "Valkey"},
	{Driver: constants.DriverMongo, Family: constants.FamilyMongo, Label: "MongoDB"},
}

// Drivers serves the driver catalog.
type Drivers struct{ usecases.RegistryDeps }

// @Summary List the drivers the console can register
// @Success 200 {array} entity.DriverInfo
// @Router /external/v1/config/drivers [get]
func (uc Drivers) Serve(w http.ResponseWriter, r *http.Request) {
	_, end := logging.StartSpan(r.Context())
	defer end(nil)
	uc.Respond(w, driverCatalog)
}
