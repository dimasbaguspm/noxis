package constants

// Metric catalog. Every metric name and label key is registered here so the call sites and the (separate) collector task agree on one spelling.

// MetricKind is the instrument a metric is recorded with.
type MetricKind string

const (
	MetricCounter   MetricKind = "counter"
	MetricGauge     MetricKind = "gauge"
	MetricHistogram MetricKind = "histogram"
)

// Metric name constants.
const (
	MetricConnectionsTotal = "connections_total"
	MetricChangesSubmitted = "changes_submitted_total"
	MetricChangesApplied   = "changes_applied_total"
	MetricChangesFailed    = "changes_failed_total"
	MetricQueryDuration    = "query_duration_seconds"
	MetricUsecaseErrors    = "usecase_errors_total"
	MetricSchedulerTicks   = "scheduler_ticks_total"
)

// Metric label keys.
const (
	LabelFamily  = "family"
	LabelDriver  = "driver"
	LabelEngine  = "engine"
	LabelOp      = "operation"
	LabelStatus  = "status"
	LabelUsecase = "usecase"
)

// Metric is one registered metric definition: the name, its instrument and the label keys a sample carries.
type Metric struct {
	Name   string
	Kind   MetricKind
	Help   string
	Labels []string
}

// Metrics catalogues every metric noxis records.
var Metrics = []Metric{
	{Name: MetricConnectionsTotal, Kind: MetricGauge, Help: "Registered connections.", Labels: []string{LabelFamily, LabelDriver}},
	{Name: MetricChangesSubmitted, Kind: MetricCounter, Help: "Changes proposed.", Labels: []string{LabelFamily, LabelEngine}},
	{Name: MetricChangesApplied, Kind: MetricCounter, Help: "Changes applied.", Labels: []string{LabelFamily}},
	{Name: MetricChangesFailed, Kind: MetricCounter, Help: "Changes that failed validation or apply.", Labels: []string{LabelFamily, LabelStatus}},
	{Name: MetricQueryDuration, Kind: MetricHistogram, Help: "Console op duration in seconds.", Labels: []string{LabelEngine, LabelOp}},
	{Name: MetricUsecaseErrors, Kind: MetricCounter, Help: "Use-case errors by outcome.", Labels: []string{LabelUsecase, LabelStatus}},
	{Name: MetricSchedulerTicks, Kind: MetricCounter, Help: "External scheduler passes.", Labels: []string{LabelStatus}},
}
