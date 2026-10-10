// Package modules contains the engine-family adapters (SQL, memory-cache,
// MongoDB) and the registry that names them for the bootstrap dependency
// injection. Each family implements interfaces.Module; modules.go is the only
// place that lists every built-in adapter, so wiring imports just this
// package.
package modules

import (
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
)

// Default builds the built-in engine-family registry. Driver lookups
// (moduleByDriver) and /v1/config/drivers resolve through this map.
func Default() map[string]interfaces.Module {
	return map[string]interfaces.Module{
		constants.FamilySQL:      sql.New(),
		constants.FamilyMemcache: memcache.New(),
		constants.FamilyMongo:    mongodb.New(),
	}
}
