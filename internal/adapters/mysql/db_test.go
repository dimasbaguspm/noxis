package mysql

import (
	"context"
	"errors"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// target is one governed server as the adapter sees it: an address, a schema, an account and the
// mode its connection was declared with.
func target() domain.Target {
	return domain.Target{
		Driver: constants.DriverMySQL, Host: "127.0.0.1", Port: 3306,
		Database: "app", Username: "noxis", TLSMode: constants.TLSPreferred,
	}
}

// One Go driver serves both engines, so the name it is opened under must be the driver's, not the
// engine's. A build that opened "mariadb" would fail at connect time with an unknown driver, which is
// the whole reason a target declares an engine and the adapter does not trust it.
func TestOneDriverServesBothEngines(t *testing.T) {
	if driverName != constants.DriverMySQL {
		t.Fatalf("driverName = %q, want the one registered name %q", driverName, constants.DriverMySQL)
	}
	for _, engine := range constants.Drivers {
		t.Run(engine, func(t *testing.T) {
			if !constants.IsDriver(engine) {
				t.Fatalf("%q was declared and then refused", engine)
			}
			declared := target()
			declared.Driver = engine
			cfg, err := mysqldriver.ParseDSN(dsnOf(declared, "pw"))
			if err != nil {
				t.Fatalf("the DSN for %q does not parse: %v", engine, err)
			}
			if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:3306" {
				t.Fatalf("%q dialled %s://%s, want tcp://127.0.0.1:3306", engine, cfg.Net, cfg.Addr)
			}
		})
	}
	// Any other engine name is refused before a connection is ever built.
	if constants.IsDriver("postgres") {
		t.Fatal("an engine this build does not speak was accepted")
	}
}

// The probe ceiling is a ceiling, not a request: a deployment that forgot to set one still gets a
// bounded probe rather than a request that hangs on a server that never answers.
func TestNewBoundsTheProbe(t *testing.T) {
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"a configured ceiling is kept", 3 * time.Second, 3 * time.Second},
		{"zero falls back to the default", 0, 5 * time.Second},
		{"a negative ceiling falls back to the default", -time.Second, 5 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := New(tc.timeout).probeTimeout; got != tc.want {
				t.Fatalf("probeTimeout = %v, want %v", got, tc.want)
			}
		})
	}
}

// A change aimed at another schema is refused before the adapter opens a connection: the verdict is
// stored as unsupported, the error names both schemas, and the note says plainly that nothing ran, so
// a reviewer is never shown evidence for a statement that never reached a server.
func TestRehearseRefusesAnotherSchemaWithoutConnecting(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change domain.Change
		want   domain.Tier
	}{
		{
			name:   "no tier asked for defaults to the shadow one",
			change: domain.Change{Query: "ALTER TABLE billing.invoices ADD COLUMN note TEXT"},
			want:   domain.TierShadow,
		},
		{
			name:   "the tier asked for is the one reported",
			change: domain.Change{Query: "ALTER TABLE billing.invoices ADD COLUMN note TEXT", Tier: domain.TierStatic},
			want:   domain.TierStatic,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := New(time.Second).Rehearse(context.Background(), target(), "pw", tc.change)
			if err != nil {
				t.Fatalf("Rehearse returned an error instead of a verdict: %v", err)
			}
			if got.Tier != tc.want {
				t.Fatalf("tier = %q, want %q", got.Tier, tc.want)
			}
			if got.Status != domain.RehearsalUnsupported {
				t.Fatalf("status = %q, want %q", got.Status, domain.RehearsalUnsupported)
			}
			for _, schema := range []string{"billing", "app"} {
				if !strings.Contains(got.Error, schema) {
					t.Fatalf("error %q does not name schema %q", got.Error, schema)
				}
			}
			if got.Note == "" {
				t.Fatal("the verdict has no note saying nothing was run")
			}
			if got.SchemaDiff != "" || got.Plan != "" || got.RowDiff != "" {
				t.Fatalf("a rehearsal that never connected produced evidence: %+v", got)
			}
			if got.CreatedAt.IsZero() {
				t.Fatal("the verdict has no timestamp")
			}
		})
	}
}

// A static rehearsal parses, plans and estimates, and it can never claim more than partial: nothing was
// executed, so nothing was proven. A change that names no table has nothing to estimate, so the whole
// check finishes without a question being put to the server, and the note has to say both things by
// name rather than leave a reader to infer them.
func TestStaticRehearsalWithoutATable(t *testing.T) {
	change := domain.Change{Query: "CREATE DATABASE analytics", Tier: domain.TierStatic}
	got, err := New(time.Second).Rehearse(context.Background(), target(), "pw", change)
	if err != nil {
		t.Fatalf("Rehearse: %v", err)
	}
	if got.Status != domain.RehearsalPartial {
		t.Fatalf("status = %q, want %q: nothing was executed", got.Status, domain.RehearsalPartial)
	}
	if got.Tier != domain.TierStatic {
		t.Fatalf("tier = %q, want %q", got.Tier, domain.TierStatic)
	}
	if !got.Irreversible {
		t.Fatal("a change with no rollback was reported as reversible")
	}
	for _, want := range []string{"nothing was executed", "no rollback was given"} {
		if !strings.Contains(got.Note, want) {
			t.Fatalf("note %q does not say %q", got.Note, want)
		}
	}
	for _, unwanted := range []string{"EXPLAIN unavailable", "no estimate"} {
		if strings.Contains(got.Note, unwanted) {
			t.Fatalf("note %q claims a failed query: nothing was asked of a server", got.Note)
		}
	}
	if got.SchemaDiff != "" {
		t.Fatalf("schema diff = %q, want none: no schema was rebuilt", got.SchemaDiff)
	}
}

// Sizing a change that names no table is not an error and not a query: there is nothing to estimate,
// so the plan comes back empty and the caller decides for itself. This is the path a DDL takes, which
// is the common case.
func TestPlanOfAStatementNamingNoTable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query string
	}{
		{"a statement nobody can read", ""},
		{"a select that names no table", "SELECT 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := New(time.Second).Plan(context.Background(), target(), "pw", domain.Change{Query: tc.query})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			if plan != (domain.Plan{}) {
				t.Fatalf("plan = %+v, want the zero plan", plan)
			}
		})
	}
}

// Apply classifies before it connects, so a statement it cannot read is reported as unsupported and
// refunded without a server being dialled. The password must not come back in the message.
func TestApplyRefusesAnUnreadableStatement(t *testing.T) {
	_, err := New(time.Second).Apply(context.Background(), target(), "s3cret", domain.Change{Query: "  "}, nil)
	if !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("Apply error = %v, want it to wrap ErrUnsupported", err)
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("the password survived into %v", err)
	}
}

// staticPlan is skipped for DDL before the engine is asked, because DDL has no plan and asking would
// turn a fact about the engine into a failed rehearsal. The pool is nil here on purpose: reaching it
// would panic, so passing proves no query was attempted.
func TestStaticPlanSkipsDDLWithoutAsking(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
	}{
		{"a DDL", "ALTER TABLE users ADD COLUMN note TEXT"},
		{"a DDL that creates a table", "CREATE TABLE users (id int)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := staticPlan(context.Background(), nil, tc.statement)
			if err != nil {
				t.Fatalf("staticPlan asked the engine about DDL: %v", err)
			}
			if plan != "" {
				t.Fatalf("plan = %q, want none for DDL", plan)
			}
		})
	}
}
