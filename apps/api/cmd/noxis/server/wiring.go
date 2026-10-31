package server

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	authsvc "noxis/internal/auth"
	"noxis/internal/common/config"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/repositories/sqlite"
	"noxis/internal/usecases"
)

// Wiring is what the route table needs to build a handler: the framework every use case embeds, the module registry the
// request context carries, and the console's auth bootstrap. There is no struct of named handlers — the route table names
// them where it declares them, so adding an endpoint touches one file.
type Wiring struct {
	Usecases usecases.RegistryDeps
	Modules  modules.Registry
	Auth     entity.AuthConfig
}

// NewWiring assembles the runtime container once: the store's repositories, the driver registry, and the auth service.
func NewWiring(store *sqlite.Store, cfg config.Config, authSvc *authsvc.Service) Wiring {
	// Instruments come from the global meter provider the logger setup registered; without a collector every instrument
	// is a no-op.
	m, err := metrics.New(otel.Meter(metrics.Scope))
	if err != nil {
		slog.Error("metric instruments", "error", err)
	}
	mods := modules.Default()
	repos := repositories.Default(store)
	registerConnectionGauge(m, repos.Connections)
	return Wiring{
		Modules: mods,
		Auth:    cfg.AuthInfo(),
		Usecases: usecases.RegistryDeps{
			Repositories: repos,
			Modules:      mods,
			Auth:         authSvc,
			AccessTTL:    cfg.GrantTTL(),
			Metrics:      m,
		},
	}
}

// registerConnectionGauge reports connections per driver from the repository on the collector's cadence.
func registerConnectionGauge(m *metrics.Metrics, connections interfaces.ConnectionRepository) {
	if m == nil {
		return
	}
	_, err := otel.Meter(metrics.Scope).RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		list, err := connections.List(ctx)
		if err != nil {
			return err
		}
		counts := map[string]int64{}
		for i := range list {
			counts[list[i].Driver]++
		}
		for driver, n := range counts {
			o.ObserveInt64(m.Connections, n, metric.WithAttributes(attribute.String(constants.LabelDriver, driver)))
		}
		return nil
	}, m.Connections)
	if err != nil {
		slog.Error("connection gauge", "error", err)
	}
}
