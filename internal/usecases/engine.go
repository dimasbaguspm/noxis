package usecases

import (
	"fmt"

	"noxis/internal/interfaces"
)

// Engine resolves the module named by an engine family through the registry.
// Handlers pass the request's {engine} path parameter; the instance workflows
// resolve by driver via moduleByDriver.
func (a *App) Engine(name string) (interfaces.Module, error) {
	mod, ok := a.Modules[name]
	if !ok {
		return nil, fmt.Errorf("%w: no engine %q", interfaces.ErrUnsupported, name)
	}
	return mod, nil
}

// EngineSQL, EngineMongo and EngineMemcache resolve an engine family and
// narrow it to the capability the caller needs; a family that does not serve
// it is unsupported (404).
func (a *App) EngineSQL(name string) (interfaces.SQLModule, error) {
	mod, err := a.Engine(name)
	if err != nil {
		return nil, err
	}
	return sqlOf(mod)
}

func (a *App) EngineMongo(name string) (interfaces.MongoModule, error) {
	mod, err := a.Engine(name)
	if err != nil {
		return nil, err
	}
	return mongoOf(mod)
}

func (a *App) EngineMemcache(name string) (interfaces.MemcacheModule, error) {
	mod, err := a.Engine(name)
	if err != nil {
		return nil, err
	}
	return memcacheOf(mod)
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
