// Package modules contains the engine-family adapters (SQL, memory-cache,
// MongoDB) and the registry that names them. Registry is a single reference
// interface every layer receives — passed explicitly, so a use case resolves
// an engine from a dependency, never from the request context.
package modules

import (
	"fmt"

	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/auth"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
)

// Registry resolves an engine family by its {engine} name and by driver, and
// carries the process auth service. It is the one handle to the built-in
// adapters and services; nothing else declares a module map.
type Registry interface {
	// Engine resolves the module named by an engine family. A family that is
	// not registered is unsupported (404).
	Engine(name string) (interfaces.Module, error)
	EngineSQL(name string) (interfaces.SQLModule, error)
	EngineMongo(name string) (interfaces.MongoModule, error)
	EngineMemcache(name string) (interfaces.MemcacheModule, error)
	// Serves reports the family that serves a driver, rather than a
	// hardcoded driver-to-family switch.
	Serves(driver string) (interfaces.Module, bool)
	// Auth is the process auth service: local accounts plus OIDC.
	Auth() *auth.Service
}

// New wraps a family map and the auth service in a registry (Default builds the
// built-in set).
func New(mods map[string]interfaces.Module, authSvc *auth.Service) Registry {
	return registry{mods: mods, auth: authSvc}
}

// Default builds the built-in engine-family registry with the auth service.
func Default(authSvc *auth.Service) Registry {
	return New(map[string]interfaces.Module{
		constants.FamilySQL:      sql.New(),
		constants.FamilyMemcache: memcache.New(),
		constants.FamilyMongo:    mongodb.New(),
	}, authSvc)
}

type registry struct {
	mods map[string]interfaces.Module
	auth *auth.Service
}

func (r registry) Auth() *auth.Service { return r.auth }

func (r registry) Engine(name string) (interfaces.Module, error) {
	mod, ok := r.mods[name]
	if !ok {
		return nil, fmt.Errorf("%w: no engine %q", interfaces.ErrUnsupported, name)
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
	return fmt.Errorf("%w: engine %q does not serve %s", interfaces.ErrUnsupported, mod.Family(), what)
}
