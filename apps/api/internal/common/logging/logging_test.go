package logging

import (
	"log/slog"
	"testing"

	otellog "go.opentelemetry.io/otel/log"
)

// TestSignalURL pins the forms NOXIS_OTEL_COLLECTOR accepts: a full URL (https
// enables TLS) and a bare host:port, which stays plaintext. A URL with no path
// gets the signal's own path; one that carries a path keeps it.
func TestSignalURL(t *testing.T) {
	cases := []struct {
		collector string
		path      string
		want      string
	}{
		{"collector:4318", pathLogs, "http://collector:4318/v1/logs"},
		{"http://collector:4318", pathTraces, "http://collector:4318/v1/traces"},
		{"https://otel.dimasbaguspm.dev", pathTraces, "https://otel.dimasbaguspm.dev/v1/traces"},
		{"https://otel.dimasbaguspm.dev", pathLogs, "https://otel.dimasbaguspm.dev/v1/logs"},
		{"https://otel.example.com/", pathTraces, "https://otel.example.com/v1/traces"},
		{"https://otel.example.com/custom", pathTraces, "https://otel.example.com/custom"},
	}
	for _, c := range cases {
		if got := signalURL(c.collector, c.path); got != c.want {
			t.Errorf("signalURL(%q, %q) = %q, want %q", c.collector, c.path, got, c.want)
		}
	}
}

// TestSeverity maps every slog level onto the OTel scale.
func TestSeverity(t *testing.T) {
	cases := map[slog.Level]otellog.Severity{
		slog.LevelDebug: otellog.SeverityDebug,
		slog.LevelInfo:  otellog.SeverityInfo,
		slog.LevelWarn:  otellog.SeverityWarn,
		slog.LevelError: otellog.SeverityError,
	}
	for in, want := range cases {
		if got := severity(in); got != want {
			t.Errorf("severity(%v) = %v, want %v", in, got, want)
		}
	}
}

// TestFlatten prefixes the group path and keeps the attribute types.
func TestFlatten(t *testing.T) {
	kvs := flatten([]string{"http"}, slog.String("method", "GET"), slog.Int("status", 500))
	if len(kvs) != 2 {
		t.Fatalf("got %d attrs, want 2", len(kvs))
	}
	if kvs[0].Key != "http.method" || kvs[0].Value.AsString() != "GET" {
		t.Errorf("first = %+v", kvs[0])
	}
	if kvs[1].Key != "http.status" || kvs[1].Value.AsInt64() != 500 {
		t.Errorf("second = %+v", kvs[1])
	}
}

// TestFlattenNestedGroup recurses into a group attribute.
func TestFlattenNestedGroup(t *testing.T) {
	kvs := flatten(nil, slog.Group("db", slog.String("system", "postgres")))
	if len(kvs) != 1 || kvs[0].Key != "db.system" {
		t.Fatalf("got %+v", kvs)
	}
}

