// Package modules contains engine adapters and the explicit registry use cases receive.
package modules

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"noxis/internal/common"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
	"noxis/internal/modules/cockroach"
	"noxis/internal/modules/couchdb"
	"noxis/internal/modules/dragonfly"
	"noxis/internal/modules/elasticsearch"
	"noxis/internal/modules/mariadb"
	"noxis/internal/modules/meilisearch"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/mysql"
	"noxis/internal/modules/postgres"
	"noxis/internal/modules/redis"
	"noxis/internal/modules/splunk"
	"noxis/internal/modules/sqlite"
	"noxis/internal/modules/valkey"
)

// Registry loads a typed module from the request context and resolves a driver to the module that serves it.
type Registry interface {
	common.ModuleLoader
	// Engine resolves the module registered for a driver name.
	Engine(driver string) (interfaces.Module, error)
	// Drivers lists the registered driver names in a stable order.
	Drivers() []string
}

// New wraps a driver-keyed module map in a registry (Default builds the built-in set).
func New(mods map[string]interfaces.Module) Registry {
	return registry{mods: mods}
}

// Default builds the built-in registry: one module per driver name, so nothing else declares what the process can do.
func Default() Registry {
	// The cache drivers are separate modules, not one module holding a driver field: valkey is a state, not an alias.
	// The same holds for every entry here — a module names its own driver and refuses the rest.
	built := []interfaces.Module{
		postgres.New(), mysql.New(), mariadb.New(), sqlite.New(), cockroach.New(),
		redis.New(), valkey.New(), dragonfly.New(),
		mongodb.New(), couchdb.New(),
		elasticsearch.New(), meilisearch.New(), splunk.New(),
	}
	mods := make(map[string]interfaces.Module, len(built))
	for _, module := range built {
		mods[module.Driver()] = module
	}
	return New(mods)
}

type registry struct {
	mods map[string]interfaces.Module
}

// LoadModule fills a requested capability from the registry, so a handler names the capability it needs rather than a
// concrete module.
func (r registry) LoadModule(ctx context.Context, target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.New("nil module target")
	}
	capability := value.Elem().Type()
	if capability.Kind() != reflect.Interface {
		return fmt.Errorf("%w: no module registered for %T", constants.ErrUnsupported, target)
	}
	driver, ok := common.ModuleDriver(ctx)
	if !ok {
		return fmt.Errorf("%w: driver missing from context", constants.ErrUnsupported)
	}
	module, ok := r.mods[driver]
	if !ok {
		return fmt.Errorf("%w: no engine serves driver %q", constants.ErrUnsupported, driver)
	}
	if !reflect.TypeOf(module).Implements(capability) {
		return fmt.Errorf("%w: driver %q does not serve %s", constants.ErrUnsupported, driver, capabilityName(capability))
	}
	value.Elem().Set(reflect.ValueOf(module))
	return nil
}

// capabilityName is the interface name without its package, so the refusal reads "does not serve SchemaReader".
func capabilityName(t reflect.Type) string {
	name := t.String()
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

// Engine resolves the module registered for a driver name.
func (r registry) Engine(driver string) (interfaces.Module, error) {
	module, ok := r.mods[driver]
	if !ok || module == nil {
		return nil, fmt.Errorf("%w: no engine serves driver %q", constants.ErrUnsupported, driver)
	}
	return module, nil
}

// Drivers lists the registered driver names, sorted so a caller that walks them is deterministic.
func (r registry) Drivers() []string {
	drivers := make([]string, 0, len(r.mods))
	for driver := range r.mods {
		drivers = append(drivers, driver)
	}
	sort.Strings(drivers)
	return drivers
}
