// Package modules contains the engine-family adapters (SQL, memory-cache, MongoDB) and the registry that names them. Registry is a single reference interface every layer receives — passed explicitly, so a use case resolves an engine from a dependency, never from the request context.
package modules

import (
	"fmt"

	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/auth"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
)

// Registry resolves an engine family by its {engine} name and by driver, and carries the process auth service.
type Registry interface {
	// Engine resolves the module named by an engine family.
	Engine(name string) (interfaces.Module, error)
	EngineSQL(name string) (interfaces.SQLModule, error)
	EngineMongo(name string) (interfaces.MongoModule, error)
	EngineMemcache(name string) (interfaces.MemcacheModule, error)
	// Serves reports the family that serves a driver, rather than a hardcoded driver-to-family switch.
	Serves(driver string) (interfaces.Module, bool)
	// Auth is the process auth service: local accounts plus OIDC.
	Auth() *auth.Service
	// Changes is every family's change queue, so a use case reaches the flow it serves the same way it reaches an engine.
	Changes() interfaces.ChangeFlows
}

// New wraps a family map, the change queues and the auth service in a registry (Default builds the built-in set).
func New(mods map[string]interfaces.Module, changes interfaces.ChangeFlows, authSvc *auth.Service) Registry {
	return registry{mods: mods, changes: changes, auth: authSvc}
}

// Default builds the built-in registry: every engine family, its change queue and the auth service, so nothing else declares what the process can do.
func Default(connections interfaces.ConnectionRepository, store interfaces.ChangeRepository, m *metrics.Metrics, authSvc *auth.Service) Registry {
	sqlDB, mongoDB := sql.New(), mongodb.New()
	return New(
		map[string]interfaces.Module{
			constants.FamilySQL:      sqlDB,
			constants.FamilyMemcache: memcache.New(),
			constants.FamilyMongo:    mongoDB,
		},
		interfaces.ChangeFlows{
			SQL:   sql.NewChanges(connections, store, sqlDB, m),
			Mongo: mongodb.NewChanges(connections, store, mongoDB, m),
		},
		authSvc,
	)
}

type registry struct {
	mods    map[string]interfaces.Module
	changes interfaces.ChangeFlows
	auth    *auth.Service
}

func (r registry) Auth() *auth.Service { return r.auth }

func (r registry) Changes() interfaces.ChangeFlows { return r.changes }

func (r registry) Engine(name string) (interfaces.Module, error) {
	mod, ok := r.mods[name]
	if !ok {
		return nil, fmt.Errorf("%w: no engine %q", constants.ErrUnsupported, name)
	}
	return mod, nil
}

func (r registry) EngineSQL(name string) (interfaces.SQLModule, error) {
	mod, err := r.Engine(name)
	if err != nil {
		return nil, err
	}
	return sqlOf(mod)
}

func (r registry) EngineMongo(name string) (interfaces.MongoModule, error) {
	mod, err := r.Engine(name)
	if err != nil {
		return nil, err
	}
	return mongoOf(mod)
}

func (r registry) EngineMemcache(name string) (interfaces.MemcacheModule, error) {
	mod, err := r.Engine(name)
	if err != nil {
		return nil, err
	}
	return memcacheOf(mod)
}

func (r registry) Serves(driver string) (interfaces.Module, bool) {
	for _, mod := range r.mods {
		if mod.Supports(driver) {
			return mod, true
		}
	}
	return nil, false
}

func sqlOf(mod interfaces.Module) (interfaces.SQLModule, error) {
	m, ok := mod.(interfaces.SQLModule)
	if !ok {
		return nil, unsupported(mod, "sql operations")
	}
	return m, nil
}

func mongoOf(mod interfaces.Module) (interfaces.MongoModule, error) {
	m, ok := mod.(interfaces.MongoModule)
	if !ok {
		return nil, unsupported(mod, "mongodb operations")
	}
	return m, nil
}

func memcacheOf(mod interfaces.Module) (interfaces.MemcacheModule, error) {
	m, ok := mod.(interfaces.MemcacheModule)
	if !ok {
		return nil, unsupported(mod, "memory-cache operations")
	}
	return m, nil
}

func unsupported(mod interfaces.Module, what string) error {
	return fmt.Errorf("%w: engine %q does not serve %s", constants.ErrUnsupported, mod.Family(), what)
}
