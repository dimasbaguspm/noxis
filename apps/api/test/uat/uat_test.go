package uat

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestUAT runs the Gherkin acceptance suite against the real compose stack (noxis + postgres + redis on host docker).
func TestUAT(t *testing.T) {
	tags := strings.TrimSpace(os.Getenv("UAT_TAGS"))
	if tags == "" {
		tags = "~@slow"
	}
	// Workers run features in parallel; the default stays serial because the cache features share one redis database,
	// so a flush in one feature empties the keys another is browsing. Raise it with UAT_CONCURRENCY once those own their database.
	concurrency := 1
	if raw := strings.TrimSpace(os.Getenv("UAT_CONCURRENCY")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("UAT_CONCURRENCY: %v", err)
		}
		concurrency = n
	}

	suite := godog.TestSuite{
		Name: "noxis",
		Options: &godog.Options{
			Format:      "pretty",
			Paths:       []string{"features"},
			Tags:        tags,
			TestingT:    t,
			Randomize:   -1, // keep the serialized, deterministic order
			Concurrency: concurrency,
			// Strict: an undefined or pending step fails the suite. Without it godog skips the step and reports a green run, which is how a mistyped step expression once went unnoticed.
			Strict: true,
		},
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			ctx.BeforeSuite(func() {
				if envTrue("UAT_STUB") {
					// Offline replay: the compose file is still the source for the base URL and targets, but no stack comes up.
					if err := loadTargets(); err != nil {
						t.Fatalf("load targets: %v", err)
					}
					return
				}
				if err := stackUp(); err != nil {
					t.Fatalf("stack up: %v", err)
				}
				if err := login(); err != nil {
					t.Fatalf("login: %v", err)
				}
				if err := pruneConnections(); err != nil {
					t.Fatalf("prune connections: %v", err)
				}
			})
			ctx.AfterSuite(func() {
				if envTrue("UAT_STUB") {
					return
				}
				stackDown()
			})
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			(&world{}).steps(ctx)
		},
	}
	if suite.Run() != 0 {
		t.Fail()
	}
}
