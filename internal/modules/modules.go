// Package modules contains the engine-family adapters (SQL, memory-cache,
// MongoDB) and the registry that names them. Registry is a single reference
// interface every layer receives — embedded in the base request context at
// boot, so any use case can resolve an engine from the ctx it is handed.
package modules

import (
	"context"
	"fmt"

	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
)

// Registry resolves an engine family by its {engine} name and by driver. It
// is the one handle to the built-in adapters; nothing else declares a module
// map.
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
}

// New wraps a family map in a registry (Default builds the built-in set).
func New(mods map[string]interfaces.Module) Registry { return registry{mods: mods} }

// Default builds the built-in engine-family registry.
func Default() Registry {
	return New(map[string]interfaces.Module{
		constants.FamilySQL:      sql.New(),
		constants.FamilyMemcache: memcache.New(),
		constants.FamilyMongo:    mongodb.New(),
	})
}

type registry struct{ mods map[string]interfaces.Module }

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

type registryKey struct{}

// WithRegistry returns a context carrying the boot-time registry, so any use
// case can resolve an engine from the ctx it is handed.
func WithRegistry(ctx context.Context, r Registry) context.Context {
	return context.WithValue(ctx, registryKey{}, r)
}

// RegistryFrom returns the registry carried by the context. It panics when
// the context was not built by WithRegistry: engine resolution without the
// registry is a wiring bug and must surface, not nil-dereference.
func RegistryFrom(ctx context.Context) Registry {
	r, ok := ctx.Value(registryKey{}).(Registry)
	if !ok {
		panic("modules.RegistryFrom: registry missing from context (seed with WithRegistry at boot)")
	}
	return r
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
