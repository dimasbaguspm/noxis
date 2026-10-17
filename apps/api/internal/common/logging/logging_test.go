package logging

import "testing"

// TestEndpointURL pins the forms NOXIS_OTEL_COLLECTOR accepts: a full URL
// (https enables TLS) and a bare host:port, which stays plaintext. A URL with
// no path gets the traces path, one that carries its own keeps it.
func TestEndpointURL(t *testing.T) {
	cases := map[string]string{
		"collector:4318":                     "http://collector:4318/v1/traces",
		"http://collector:4318":              "http://collector:4318/v1/traces",
		"https://otel.dimasbaguspm.dev":      "https://otel.dimasbaguspm.dev/v1/traces",
		"https://collector.example:4318":     "https://collector.example:4318/v1/traces",
		"https://otel.example.com/":          "https://otel.example.com/v1/traces",
		"https://otel.example.com/v1/traces": "https://otel.example.com/v1/traces",
		"https://otel.example.com/custom":    "https://otel.example.com/custom",
	}
	for in, want := range cases {
		if got := endpointURL(in); got != want {
			t.Errorf("endpointURL(%q) = %q, want %q", in, got, want)
		}
	}
}
