package metadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"noxis/internal/usecases"
)

// Only an https collector reaches the console; anything else means the browser
// keeps tracing off and an internal host never leaks.
func TestTracingExposesOnlyHTTPS(t *testing.T) {
	cases := map[string]string{
		"https://otel.example.com":     "https://otel.example.com",
		"https://otel.example.com:443": "https://otel.example.com:443",
		"http://otel.example.com:4317": "",
		"otel.example.com:4317":        "",
		"":                             "",
	}
	for in, want := range cases {
		rec := httptest.NewRecorder()
		Tracing{RegistryDeps: usecases.RegistryDeps{}, Endpoint: in}.
			Serve(rec, httptest.NewRequest(http.MethodGet, "/external/v1/config/tracing", http.NoBody))

		var got struct {
			Endpoint string `json:"endpoint"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode %q: %v", in, err)
		}
		if got.Endpoint != want {
			t.Errorf("endpoint(%q) = %q, want %q", in, got.Endpoint, want)
		}
	}
}
