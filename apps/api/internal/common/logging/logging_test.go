package logging

import "testing"

// TestEndpointURL pins the two forms NOXIS_OTEL_COLLECTOR accepts: a full URL
// (https enables TLS) and the legacy bare host:port, which stays plaintext.
func TestEndpointURL(t *testing.T) {
	cases := map[string]string{
		"collector:4317":                 "http://collector:4317",
		"http://collector:4317":          "http://collector:4317",
		"https://otel.dimasbaguspm.dev":  "https://otel.dimasbaguspm.dev",
		"https://collector.example:4317": "https://collector.example:4317",
	}
	for in, want := range cases {
		if got := endpointURL(in); got != want {
			t.Errorf("endpointURL(%q) = %q, want %q", in, got, want)
		}
	}
}
