package uat

import (
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestUAT runs the Gherkin acceptance suite against the real compose stack
// (Kafka + master + slave on host docker). Default run excludes @slow
// scenarios (image pulls); set UAT_TAGS to override, e.g.
// UAT_TAGS="~@slow" make uat.
func TestUAT(t *testing.T) {
	tags := strings.TrimSpace(os.Getenv("UAT_TAGS"))
	if tags == "" {
		tags = "~@slow"
	}

	suite := godog.TestSuite{
		Name: "noxis",
		Options: &godog.Options{
			Format:    "pretty",
			Paths:     []string{"features"},
			Tags:      tags,
			TestingT:  t,
			Randomize: -1, // keep the serialized, deterministic order
		},
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			ctx.BeforeSuite(func() {
				if err := stackUp(); err != nil {
					t.Fatalf("stack up: %v", err)
				}
			})
			ctx.AfterSuite(func() {
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
