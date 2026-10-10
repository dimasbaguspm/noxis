package api

import (
	"context"
	"net/http"

	"noxis/internal/constants"
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
	{constants.DriverRedis, constants.FamilyMemcache, "Redis"},
	{constants.DriverValkey, constants.FamilyMemcache, "Valkey"},
	{constants.DriverMongo, constants.FamilyMongo, "MongoDB"},
}

// @Summary List the drivers the console can register
// @Success 200 {array} api.DriverInfo
// @Router /v1/config/drivers [get]
func (s *Server) ListDrivers(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, func(context.Context) (any, error) {
		return driverCatalog, nil
	})
}
