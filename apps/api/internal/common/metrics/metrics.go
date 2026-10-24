// Package metrics holds the instruments the API records into. The catalog in the constants package is the single source of truth — every name, kind and label key comes from there, so a call site and the catalog cannot drift.
package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"noxis/internal/constants"
)

// Scope names this instrumentation in the exported data.
const Scope = "noxis"

// Metrics holds one instrument per catalog entry.
type Metrics struct {
	Connections    metric.Int64ObservableGauge
	ChangesSubmit  metric.Int64Counter
	ChangesApplied metric.Int64Counter
	ChangesFailed  metric.Int64Counter
	QueryDuration  metric.Float64Histogram
	UsecaseErrors  metric.Int64Counter
	SchedulerTicks metric.Int64Counter

	// built records which catalog entries got an instrument, so a test can prove none was forgotten.
	built []string
}

// New builds every instrument the catalog declares.
func New(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}
	steps := []struct {
		name string
		run  func() error
	}{
		{constants.MetricConnectionsTotal, func() (err error) {
			m.Connections, err = meter.Int64ObservableGauge(constants.MetricConnectionsTotal, metric.WithDescription(help(constants.MetricConnectionsTotal)))
			return
		}},
		{constants.MetricChangesSubmitted, func() (err error) {
			m.ChangesSubmit, err = meter.Int64Counter(constants.MetricChangesSubmitted, metric.WithDescription(help(constants.MetricChangesSubmitted)))
			return
		}},
		{constants.MetricChangesApplied, func() (err error) {
			m.ChangesApplied, err = meter.Int64Counter(constants.MetricChangesApplied, metric.WithDescription(help(constants.MetricChangesApplied)))
			return
		}},
		{constants.MetricChangesFailed, func() (err error) {
			m.ChangesFailed, err = meter.Int64Counter(constants.MetricChangesFailed, metric.WithDescription(help(constants.MetricChangesFailed)))
			return
		}},
		{constants.MetricQueryDuration, func() (err error) {
			m.QueryDuration, err = meter.Float64Histogram(constants.MetricQueryDuration, metric.WithDescription(help(constants.MetricQueryDuration)), metric.WithUnit("s"))
			return
		}},
		{constants.MetricUsecaseErrors, func() (err error) {
			m.UsecaseErrors, err = meter.Int64Counter(constants.MetricUsecaseErrors, metric.WithDescription(help(constants.MetricUsecaseErrors)))
			return
		}},
		{constants.MetricSchedulerTicks, func() (err error) {
			m.SchedulerTicks, err = meter.Int64Counter(constants.MetricSchedulerTicks, metric.WithDescription(help(constants.MetricSchedulerTicks)))
			return
		}},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			return nil, fmt.Errorf("metric %s: %w", s.name, err)
		}
		m.built = append(m.built, s.name)
	}
	return m, nil
}

// UsecaseError counts a use case that ended in an error, labelled with the use case and the status the caller saw.
func (m *Metrics) UsecaseError(ctx context.Context, usecase, status string) {
	if m == nil {
		return
	}
	m.UsecaseErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String(constants.LabelUsecase, usecase),
		attribute.String(constants.LabelStatus, status),
	))
}

// OpDuration records how long one console operation took.
func (m *Metrics) OpDuration(ctx context.Context, engine, op string, seconds float64) {
	if m == nil {
		return
	}
	m.QueryDuration.Record(ctx, seconds, metric.WithAttributes(
		attribute.String(constants.LabelEngine, engine),
		attribute.String(constants.LabelOp, op),
	))
}

// ChangeSubmitted counts a change that was proposed.
func (m *Metrics) ChangeSubmitted(ctx context.Context, family, engine string) {
	if m == nil {
		return
	}
	m.ChangesSubmit.Add(ctx, 1, metric.WithAttributes(
		attribute.String(constants.LabelFamily, family),
		attribute.String(constants.LabelEngine, engine),
	))
}

// ChangeApplied counts a change that was applied.
func (m *Metrics) ChangeApplied(ctx context.Context, family string) {
	if m == nil {
		return
	}
	m.ChangesApplied.Add(ctx, 1, metric.WithAttributes(attribute.String(constants.LabelFamily, family)))
}

// ChangeFailed counts a change that failed, labelled with the stage that failed.
func (m *Metrics) ChangeFailed(ctx context.Context, family, status string) {
	if m == nil {
		return
	}
	m.ChangesFailed.Add(ctx, 1, metric.WithAttributes(
		attribute.String(constants.LabelFamily, family),
		attribute.String(constants.LabelStatus, status),
	))
}

// SchedulerTick counts one scheduler pass, labelled with its outcome.
func (m *Metrics) SchedulerTick(ctx context.Context, status string) {
	if m == nil {
		return
	}
	m.SchedulerTicks.Add(ctx, 1, metric.WithAttributes(attribute.String(constants.LabelStatus, status)))
}

// help returns the catalog's description for a metric name.
func help(name string) string {
	for i := range constants.Metrics {
		if constants.Metrics[i].Name == name {
			return constants.Metrics[i].Help
		}
	}
	return ""
}
