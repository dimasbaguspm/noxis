package usecases

import (
	"context"
	"net/http"
	"time"

	"noxis/internal/common"
	"noxis/internal/common/auth"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/modules"
	"noxis/internal/repositories"
)

// RegistryDeps is embedded by every use case.
type RegistryDeps struct {
	Repositories repositories.Registry
	Modules      modules.Registry
	// Auth is the process auth service: local accounts plus OIDC.
	Auth *auth.Service
	// Metrics records the catalog's instruments. A nil value is a no-op, so a use case built by hand in a test needs no instrumentation.
	Metrics *metrics.Metrics
	// AccessTTL is how long a per-connection grant lasts. Zero falls back to constants.AccessGrantTTL, so a hand-built dep in a test still works.
	AccessTTL time.Duration
}

// RouteModule loads a typed capability for the driver the route names, so a handler never reads the engine path parameter itself.
func (d RegistryDeps) RouteModule(ctx context.Context, r *http.Request, target any) error {
	if driver := r.PathValue("driver"); driver != "" {
		ctx = common.WithModuleDriver(ctx, driver)
	}
	return common.LoadModule(ctx, target)
}

// GrantTTL is the configured grant lifetime, or the 120-day default.
func (d RegistryDeps) GrantTTL() time.Duration {
	if d.AccessTTL <= 0 {
		return constants.AccessGrantTTL
	}
	return d.AccessTTL
}
