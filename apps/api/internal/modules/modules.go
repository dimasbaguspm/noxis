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
	"noxis/internal/common/metrics"
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
	"noxis/internal/modules/sqlite"
	"noxis/internal/modules/valkey"
	"noxis/internal/repositories"
)

// Registration is one driver's engine and the queue that applies its changes; a driver with no queue leaves Changes nil.
type Registration struct {
	Engine  interfaces.Module
	Changes interfaces.ChangeFlow
}

// Registry loads a typed module from the request context and resolves a driver to the module that serves it.
type Registry interface {
	common.ModuleLoader
	// Engine resolves the module registered for a driver name.
	Engine(driver string) (interfaces.Module, error)
	// Changes resolves the change queue registered for a driver name.
	Changes(driver string) (interfaces.ChangeFlow, error)
	// Drivers lists the registered driver names in a stable order.
	Drivers() []string
}

// New wraps a driver-keyed registration map in a registry (Default builds the built-in set).
func New(mods map[string]Registration) Registry {
	return registry{mods: mods}
}

// Default builds the built-in registry: one module per driver name, with its change queue, so nothing else declares what the process can do.
func Default(repos repositories.Registry, m *metrics.Metrics) Registry {
	mods := map[string]Registration{}

	// One SQL module per driver: each serves its own driver and refuses the rest, so the registration below names no driver twice.
	pg := postgres.New()
	mods[pg.Driver()] = Registration{Engine: pg, Changes: postgres.NewChanges(repos.Connections, repos.Changes, pg, m)}

	my := mysql.New()
	mods[my.Driver()] = Registration{Engine: my, Changes: mysql.NewChanges(repos.Connections, repos.Changes, my, m)}

	maria := mariadb.New()
	mods[maria.Driver()] = Registration{Engine: maria, Changes: mariadb.NewChanges(repos.Connections, repos.Changes, maria, m)}

	lite := sqlite.New()
	mods[lite.Driver()] = Registration{Engine: lite, Changes: sqlite.NewChanges(repos.Connections, repos.Changes, lite, m)}

	cockroachDB := cockroach.New()
	mods[cockroachDB.Driver()] = Registration{Engine: cockroachDB, Changes: cockroach.NewChanges(repos.Connections, repos.Changes, cockroachDB, m)}

	// The cache drivers are separate modules, not one module holding a driver field: valkey is a state, not an alias.
	cache := redis.New()
	mods[cache.Driver()] = Registration{Engine: cache}

	valkeyCache := valkey.New()
	mods[valkeyCache.Driver()] = Registration{Engine: valkeyCache}

	df := dragonfly.New()
	mods[df.Driver()] = Registration{Engine: df}

	documents := mongodb.New()
	mods[constants.DriverMongo] = Registration{Engine: documents, Changes: mongodb.NewChanges(repos.Connections, repos.Changes, documents, m)}

	couch := couchdb.New()
	mods[couch.Driver()] = Registration{Engine: couch, Changes: couchdb.NewChanges(repos.Connections, repos.Changes, couch, m)}

	search := elasticsearch.New()
	mods[search.Driver()] = Registration{Engine: search, Changes: elasticsearch.NewChanges(repos.Connections, repos.Changes, search, m)}

	meili := meilisearch.New()
	mods[meili.Driver()] = Registration{Engine: meili, Changes: meilisearch.NewChanges(repos.Connections, repos.Changes, meili, m)}
	return New(mods)
}

type registry struct {
	mods map[string]Registration
}

// LoadModule fills a requested capability from the registry: the engine or its change queue, whichever serves it, so a new capability needs no edit here.
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
	reg, ok := r.mods[driver]
	if !ok {
		return fmt.Errorf("%w: no engine serves driver %q", constants.ErrUnsupported, driver)
	}
	// A change queue is not an engine, so the capability is checked against both halves of the registration.
	for _, candidate := range []any{reg.Engine, reg.Changes} {
		if candidate == nil || !reflect.TypeOf(candidate).Implements(capability) {
			continue
		}
		value.Elem().Set(reflect.ValueOf(candidate))
		return nil
	}
	return fmt.Errorf("%w: driver %q does not serve %s", constants.ErrUnsupported, driver, capabilityName(capability))
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
	reg, ok := r.mods[driver]
	if !ok || reg.Engine == nil {
		return nil, fmt.Errorf("%w: no engine serves driver %q", constants.ErrUnsupported, driver)
	}
	return reg.Engine, nil
}

// Changes resolves the change queue registered for a driver name; a driver without one has none rather than an empty one.
func (r registry) Changes(driver string) (interfaces.ChangeFlow, error) {
	reg, ok := r.mods[driver]
	if !ok || reg.Changes == nil {
		return nil, fmt.Errorf("%w: driver %q has no change queue", constants.ErrUnsupported, driver)
	}
	return reg.Changes, nil
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
