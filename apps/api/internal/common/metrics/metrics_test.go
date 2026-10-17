package metrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/metric/noop"

	"noxis/internal/constants"
)

// TestNewBuildsTheWholeCatalog guards the one failure this package can have: a
// catalog entry that no instrument was built for.
func TestNewBuildsTheWholeCatalog(t *testing.T) {
	m, err := New(noop.NewMeterProvider().Meter(Scope))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(m.built) != len(constants.Metrics) {
		t.Fatalf("built %d instruments for %d catalog entries", len(m.built), len(constants.Metrics))
	}
	seen := make(map[string]bool, len(m.built))
	for _, name := range m.built {
		seen[name] = true
	}
	for _, def := range constants.Metrics {
		if !seen[def.Name] {
			t.Errorf("catalog entry %q has no instrument", def.Name)
		}
	}
}

// TestNilMetricsIsSafe pins that a use case built by hand records nothing and
// never panics.
func TestNilMetricsIsSafe(t *testing.T) {
	var m *Metrics
	ctx := context.Background()
	m.UsecaseError(ctx, "x", "400")
	m.OpDuration(ctx, "sql", "query", 0.1)
	m.ChangeSubmitted(ctx, "sql", "postgres")
	m.ChangeApplied(ctx, "sql")
	m.ChangeFailed(ctx, "sql", "apply")
	m.SchedulerTick(ctx, "ok")
}
