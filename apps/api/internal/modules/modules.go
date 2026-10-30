// Package modules contains engine adapters and the explicit registry use cases receive.
package modules

import (
	"context"
	"errors"
	"fmt"

	"noxis/internal/common"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/sql"
	"noxis/internal/repositories"
)

// Registry loads a typed module from the request context and resolves a driver to the module that serves it.
type Registry interface {
	common.ModuleLoader
	// Engine resolves the module registered for a driver name.
	Engine(driver string) (interfaces.Module, error)
}

// New wraps a driver-keyed module map and the change queues in a registry (Default builds the built-in set).
func New(mods map[string]interfaces.Module, changes interfaces.ChangeFlows) Registry {
	return registry{mods: mods, changes: changes}
}

// Default builds the built-in registry: one module per driver name, with its change queue, so nothing else declares what the process can do.
func Default(repos repositories.Registry, m *metrics.Metrics) Registry {
	mods := map[string]interfaces.Module{}
	changes := interfaces.ChangeFlows{}
	for _, driver := range sqlDrivers {
		engine := sql.New(driver)
		mods[driver] = engine
		changes[driver] = sql.NewChanges(repos.Connections, repos.Changes, engine, m, driver)
	}
	for _, driver := range memcacheDrivers {
		mods[driver] = memcache.New(driver)
	}
	documents := mongodb.New()
	mods[constants.DriverMongo] = documents
	changes[constants.DriverMongo] = mongodb.NewChanges(repos.Connections, repos.Changes, documents, m)
	return New(mods, changes)
}

// sqlDrivers are the drivers the SQL engine serves, one module each.
var sqlDrivers = []string{
	constants.DriverPostgres,
	constants.DriverMySQL,
	constants.DriverMariaDB,
	constants.DriverSQLite,
	constants.DriverCockroach,
}

// memcacheDrivers are the drivers the memory-cache engine serves, one module each: valkey is its own state, not an alias.
var memcacheDrivers = []string{constants.DriverRedis, constants.DriverValkey}

type registry struct {
	mods    map[string]interfaces.Module
	changes interfaces.ChangeFlows
}

// LoadModule fills a requested capability from the registry.
func (r registry) LoadModule(ctx context.Context, target any) error {
	switch dst := target.(type) {
	case *interfaces.Module:
		return loadEngine(ctx, r, dst, "engine operations")
	case *interfaces.SQLModule:
		return loadEngine(ctx, r, dst, "sql operations")
	case *interfaces.MongoModule:
		return loadEngine(ctx, r, dst, "mongodb operations")
	case *interfaces.MemcacheModule:
		return loadEngine(ctx, r, dst, "memory-cache operations")
	case *interfaces.ChangeFlows:
		if dst == nil {
			return errors.New("nil module target")
		}
		*dst = r.changes
		return nil
	case *interfaces.ChangeFlow:
		if dst == nil {
			return errors.New("nil module target")
		}
		return r.loadFlow(ctx, func(flow interfaces.ChangeFlow) { *dst = flow })
	case *interfaces.SQLChangeFlow:
		if dst == nil {
			return errors.New("nil module target")
		}
		return r.loadFlow(ctx, func(flow interfaces.ChangeFlow) {
			if selected, ok := flow.(interfaces.SQLChangeFlow); ok {
				*dst = selected
			}
		})
	case *interfaces.MongoChangeFlow:
		if dst == nil {
			return errors.New("nil module target")
		}
		return r.loadFlow(ctx, func(flow interfaces.ChangeFlow) {
			if selected, ok := flow.(interfaces.MongoChangeFlow); ok {
				*dst = selected
			}
		})
	default:
		return fmt.Errorf("%w: no module registered for %T", constants.ErrUnsupported, target)
	}
}

// loadFlow resolves the change queue of the driver the route selected.
func (r registry) loadFlow(ctx context.Context, assign func(interfaces.ChangeFlow)) error {
	driver, ok := common.ModuleDriver(ctx)
	if !ok {
		return fmt.Errorf("%w: driver missing from context", constants.ErrUnsupported)
	}
	flow, ok := r.changes[driver]
	if !ok {
		return fmt.Errorf("%w: driver %q has no change queue", constants.ErrUnsupported, driver)
	}
	assign(flow)
	return nil
}

func loadEngine[T any](ctx context.Context, r registry, dst *T, what string) error {
	if dst == nil {
		return errors.New("nil module target")
	}
	driver, ok := common.ModuleDriver(ctx)
	if !ok {
		return fmt.Errorf("%w: driver missing from context", constants.ErrUnsupported)
	}
	module, err := r.Engine(driver)
	if err != nil {
		return err
	}
	selected, ok := module.(T)
	if !ok {
		return unsupported(module, what)
	}
	*dst = selected
	return nil
}

func (r registry) Engine(driver string) (interfaces.Module, error) {
	mod, ok := r.mods[driver]
	if !ok {
		return nil, fmt.Errorf("%w: no engine serves driver %q", constants.ErrUnsupported, driver)
	}
	return mod, nil
}

func unsupported(mod interfaces.Module, what string) error {
	return fmt.Errorf("%w: driver %q does not serve %s", constants.ErrUnsupported, mod.Driver(), what)
}
