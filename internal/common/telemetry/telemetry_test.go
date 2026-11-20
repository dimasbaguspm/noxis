package telemetry_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/config"
	"noxis/internal/common/telemetry"
	"strings"
	"testing"
	"time"
)

// A process with nowhere to send telemetry must still start, log and serve — that is the default
// deployment, and a no-op provider that panics would make the collector mandatory.
func TestNoEndpointIsANoop(t *testing.T) {
	log, shutdown, err := telemetry.New(context.Background(), config.Config{})
	if err != nil {
		t.Fatalf("New without an endpoint: %v", err)
	}
	if log == nil {
		t.Fatal("New returned no logger")
	}
	// A span with the no-op provider is still a valid context: nothing may panic.
	ctx, end := telemetry.Span(context.Background(), "noop")
	end(errors.New("irrelevant"))
	if ctx == nil {
		t.Fatal("Span returned no context")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown without providers: %v", err)
	}
}

// The proof that the wiring is real: one span, one collector, and the span arrives. A test that only
// asserts New returns a non-nil provider proves the constructors exist, not that anything leaves.
func TestSpanReachesTheCollector(t *testing.T) {
	const name = "telemetry.TestSpanReachesTheCollector"

	received := make(chan string, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), name) {
			select {
			case received <- r.URL.Path:
			default:
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")

	cfg := config.Config{OTELEndpoint: collector.URL, OTELServiceName: "noxis-test"}
	_, shutdown, err := telemetry.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New with an endpoint: %v", err)
	}

	_, end := telemetry.Span(context.Background(), name)
	end(nil)

	flush, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shutdown(flush); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	select {
	case path := <-received:
		if path != "/v1/traces" {
			t.Fatalf("span arrived at %q, want the trace endpoint", path)
		}
	case <-time.After(time.Second):
		t.Fatal("the span never reached the collector")
	}
}

// Outcome is what the apply counter is grouped by, so it has to say two things and no more.
func TestOutcome(t *testing.T) {
	if got := telemetry.Outcome(nil); got != "ok" {
		t.Fatalf("Outcome(nil) = %q, want ok", got)
	}
	if got := telemetry.Outcome(errors.New("boom")); got != "failed" {
		t.Fatalf("Outcome(err) = %q, want failed", got)
	}
}
