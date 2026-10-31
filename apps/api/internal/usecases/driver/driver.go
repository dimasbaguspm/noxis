// Package driver holds the driver-catalog use case.
package driver

import (
	"context"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/usecases"
)

// driverCatalog is the only source of the console's driver pickers: a driver is code, not data, so no UI constant can drift from the server.
var driverCatalog = []entity.DriverInfo{
	{Driver: constants.DriverPostgres, Label: "PostgreSQL"},
	{Driver: constants.DriverMySQL, Label: "MySQL"},
	{Driver: constants.DriverMariaDB, Label: "MariaDB"},
	{Driver: constants.DriverSQLite, Label: "SQLite"},
	{Driver: constants.DriverCockroach, Label: "CockroachDB"},
	{Driver: constants.DriverRedis, Label: "Redis"},
	{Driver: constants.DriverValkey, Label: "Valkey"},
	{Driver: constants.DriverDragonfly, Label: "Dragonfly"},
	{Driver: constants.DriverMongo, Label: "MongoDB"},
	{Driver: constants.DriverCouchDB, Label: "CouchDB"},
	{Driver: constants.DriverElasticsearch, Label: "Elasticsearch"},
	{Driver: constants.DriverMeilisearch, Label: "Meilisearch"},
}

// Drivers serves the driver catalog.
type Drivers struct{ usecases.RegistryDeps }

// @Summary List the drivers the console can register
// @Success 200 {array} entity.DriverInfo
// @Router /external/v1/config/drivers [get]
func (uc Drivers) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	uc.Respond(w, uc.catalog(ctx))
}

// catalog fills each entry with the capabilities its registered module implements, so the console groups its
// navigation by what a driver can do and a newly registered driver appears without a frontend edit.
func (uc Drivers) catalog(ctx context.Context) []entity.DriverInfo {
	list := make([]entity.DriverInfo, 0, len(driverCatalog))
	for _, info := range driverCatalog {
		info.Capabilities = capabilities(ctx, uc.Modules, info.Driver)
		list = append(list, info)
	}
	return list
}

// capabilities reads a driver's module and reports the surfaces it serves; an unregistered driver reports none rather than guessing.
func capabilities(ctx context.Context, registry modules.Registry, driver string) []string {
	found := []string{}
	if registry == nil {
		return found
	}
	mod, err := registry.Engine(driver)
	if err != nil {
		logging.WarnWithCtx(ctx, "driver catalog names an unregistered driver", "driver", driver, "error", err)
		return found
	}
	if _, ok := mod.(interfaces.SQLChangeEngine); ok {
		found = append(found, constants.CapabilitySQL)
	}
	if _, ok := mod.(interfaces.KeyBrowser); ok {
		found = append(found, constants.CapabilityKeys)
	}
	if _, ok := mod.(interfaces.MongoChangeEngine); ok {
		found = append(found, constants.CapabilityDocuments)
	}
	if _, ok := mod.(interfaces.SearchEngine); ok {
		found = append(found, constants.CapabilitySearch)
	}
	return found
}
