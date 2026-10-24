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

// Registry loads a typed module from the request context and maps a driver to the family that serves it.
type Registry interface {
	common.ModuleLoader
	// Engine resolves the module named by an engine family.
	Engine(name string) (interfaces.Module, error)
	// Serves reports the family that serves a driver, rather than a hardcoded driver-to-family switch.
	Serves(driver string) (interfaces.Module, bool)
}

// New wraps a family map and the change queues in a registry (Default builds the built-in set).
func New(mods map[string]interfaces.Module, changes interfaces.ChangeFlows) Registry {
	return registry{mods: mods, changes: changes}
}

// Default builds the built-in registry: every engine family, its change queue and the auth service, so nothing else declares what the process can do.
func Default(repos repositories.Registry, m *metrics.Metrics) Registry {
	sqlDB, mongoDB := sql.New(), mongodb.New()
	return New(
		map[string]interfaces.Module{
			constants.FamilySQL:      sqlDB,
			constants.FamilyMemcache: memcache.New(),
			constants.FamilyMongo:    mongoDB,
		},
		interfaces.ChangeFlows{
			SQL:   sql.NewChanges(repos.Connections, repos.Changes, sqlDB, m),
			Mongo: mongodb.NewChanges(repos.Connections, repos.Changes, mongoDB, m),
		},
	)
}

type registry struct {
	mods    map[string]interfaces.Module
	changes interfaces.ChangeFlows
}

// LoadModule fills a requested change-flow capability from the registry.
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
	case *interfaces.SQLChangeFlow:
		if dst == nil {
			return errors.New("nil module target")
		}
		if r.changes.SQL == nil {
			return fmt.Errorf("%w: no SQL change flow registered", constants.ErrUnsupported)
		}
		*dst = r.changes.SQL
		return nil
	case *interfaces.MongoChangeFlow:
		if dst == nil {
			return errors.New("nil module target")
		}
		if r.changes.Mongo == nil {
			return fmt.Errorf("%w: no Mongo change flow registered", constants.ErrUnsupported)
		}
		*dst = r.changes.Mongo
		return nil
	default:
		return fmt.Errorf("%w: no module registered for %T", constants.ErrUnsupported, target)
	}
}

func loadEngine[T any](ctx context.Context, r registry, dst *T, what string) error {
	if dst == nil {
		return errors.New("nil module target")
	}
	family, ok := common.ModuleFamily(ctx)
	if !ok {
		return fmt.Errorf("%w: engine family missing from context", constants.ErrUnsupported)
	}
	module, err := r.Engine(family)
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

func (r registry) Engine(name string) (interfaces.Module, error) {
	mod, ok := r.mods[name]
	if !ok {
		return nil, fmt.Errorf("%w: no engine %q", constants.ErrUnsupported, name)
	}
	return mod, nil
}

func (r registry) Serves(driver string) (interfaces.Module, bool) {
	for _, mod := range r.mods {
		if mod.Supports(driver) {
			return mod, true
		}
	}
	return nil, false
}

func unsupported(mod interfaces.Module, what string) error {
	return fmt.Errorf("%w: engine %q does not serve %s", constants.ErrUnsupported, mod.Family(), what)
}
