package usecases

import (
	"fmt"

	"noxis/internal/interfaces"
)

// moduleByDriver resolves the family that serves a driver through the
// registry, rather than a hardcoded driver-to-family switch.
func moduleByDriver(modules map[string]interfaces.Module, driver string) (interfaces.Module, bool) {
	for _, mod := range modules {
		if mod.Supports(driver) {
			return mod, true
		}
	}
	return nil, false
}

// Engine resolves the module named by an engine family through the registry.
// Handlers pass the request's {engine} path parameter; the scheduler and
// instance flows resolve by driver via moduleByDriver.
func Engine(modules map[string]interfaces.Module, name string) (interfaces.Module, error) {
	mod, ok := modules[name]
	if !ok {
		return nil, fmt.Errorf("%w: no engine %q", interfaces.ErrUnsupported, name)
	}
	return mod, nil
}

// EngineSQL, EngineMongo and EngineMemcache resolve an engine family and
// narrow it to the capability the caller needs; a family that does not serve
// it is unsupported (404).
func EngineSQL(modules map[string]interfaces.Module, name string) (interfaces.SQLModule, error) {
	mod, err := Engine(modules, name)
	if err != nil {
		return nil, err
	}
	return sqlOf(mod)
}

func EngineMongo(modules map[string]interfaces.Module, name string) (interfaces.MongoModule, error) {
	mod, err := Engine(modules, name)
	if err != nil {
		return nil, err
	}
	return mongoOf(mod)
}

func EngineMemcache(modules map[string]interfaces.Module, name string) (interfaces.MemcacheModule, error) {
	mod, err := Engine(modules, name)
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
