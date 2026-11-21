package mysql

import (
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
	"testing"
)

// What a probe reports is derived from the engine it measured, never from the name the target was
// declared with. Probe reads the version and the comment (probe.go:62) and hands both to
// domain.CapabilitiesFor (probe.go:63), so this pins the pair the adapter actually calls: a server
// that answers 8.0 and a server that answers 11.4 must not come back with the same notes.
func TestProbedCapabilitiesFollowTheMeasuredEngine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version string
		comment string
		flavor  string
		explain bool
		notes   []string
		silent  []string
	}{
		{
			name:    "mysql 8.0 supports everything this build offers",
			version: "8.0.36", comment: "MySQL Community Server - GPL",
			flavor: constants.DriverMySQL, explain: true,
			silent: []string{"atomic DDL", "instant DDL", "not probed yet"},
		},
		{
			name:    "mysql 8.4 supports everything this build offers",
			version: "8.4.2", comment: "MySQL Community Server - GPL",
			flavor: constants.DriverMySQL, explain: true,
			silent: []string{"atomic DDL", "not probed yet"},
		},
		{
			name:    "mysql 5.7 explains but does not do atomic DDL",
			version: "5.7.44", comment: "MySQL Community Server - GPL",
			flavor: constants.DriverMySQL, explain: true,
			notes: []string{"atomic DDL"},
		},
		{
			name:    "mariadb 10.6 explains and keeps instant DDL",
			version: "10.6.16-MariaDB", comment: "mariadb.org binary distribution",
			flavor: constants.DriverMariaDB, explain: true,
			silent: []string{"instant DDL", "atomic DDL", "not probed yet"},
		},
		{
			name:    "mariadb 11.4 explains and keeps instant DDL",
			version: "11.4.2-MariaDB", comment: "mariadb.org binary distribution",
			flavor: constants.DriverMariaDB, explain: true,
			silent: []string{"instant DDL", "not probed yet"},
		},
		{
			name:    "mariadb before 10.5 has no instant DDL",
			version: "10.3.39-MariaDB", comment: "mariadb.org binary distribution",
			flavor: constants.DriverMariaDB, explain: true,
			notes: []string{"instant DDL"},
		},
		{
			name:    "mariadb 10.0 has no JSON explain",
			version: "10.0.38-MariaDB", comment: "mariadb.org binary distribution",
			flavor: constants.DriverMariaDB, explain: false,
			notes: []string{"instant DDL"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := domain.ParseServerVersion(tc.version, tc.comment)
			caps := domain.CapabilitiesFor(server.Flavor, server)
			if caps.Flavor != tc.flavor {
				t.Fatalf("flavor = %q, want %q", caps.Flavor, tc.flavor)
			}
			if caps.ExplainJSON != tc.explain {
				t.Fatalf("explain = %v, want %v", caps.ExplainJSON, tc.explain)
			}
			// The rehearsal depends on these three, so they are true for every engine this build
			// speaks, at every version the build supports.
			if !caps.ShadowDDLRehearsal || !caps.RowImageRehearsal || !caps.TransactionalDML {
				t.Fatalf("a supported engine was denied a rehearsal: %+v", caps)
			}
			notes := strings.Join(caps.Notes, "\n")
			for _, want := range tc.notes {
				if !strings.Contains(notes, want) {
					t.Fatalf("notes %q do not mention %q", caps.Notes, want)
				}
			}
			for _, unwanted := range tc.silent {
				if strings.Contains(notes, unwanted) {
					t.Fatalf("notes %q mention %q, which is not true of this engine", caps.Notes, unwanted)
				}
			}
		})
	}
}

// A target that has not been probed yet is a target whose version is unknown, and the capabilities
// have to say so rather than assume the newest engine. The declared name is carried as declared: the
// adapter reports what it was told, and the probe is what corrects it.
func TestUnprobedCapabilitiesSaySo(t *testing.T) {
	caps := domain.CapabilitiesFor(constants.DriverMariaDB, domain.Server{Flavor: constants.DriverMariaDB})
	if caps.ExplainJSON {
		t.Fatal("EXPLAIN was promised for a server whose version nobody has read")
	}
	if !strings.Contains(strings.Join(caps.Notes, "\n"), "not probed yet") {
		t.Fatalf("notes = %v, want one saying the version is unknown", caps.Notes)
	}
}

// A declared engine that disagrees with the measured one is a fact the reviewer needs, not a typo to
// hide: the note names both, because a rehearsal planned for the wrong engine is a rehearsal of the
// wrong thing.
func TestDeclaredFlavorMismatchIsNamed(t *testing.T) {
	server := domain.ParseServerVersion("10.6.16-MariaDB", "mariadb.org binary distribution")
	caps := domain.CapabilitiesFor(constants.DriverMySQL, server)
	notes := strings.Join(caps.Notes, "\n")
	for _, want := range []string{constants.DriverMySQL, constants.DriverMariaDB} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes %q do not name %q", caps.Notes, want)
		}
	}
	// A declared name that agrees with the server, or was never declared, is not a mismatch.
	for _, declared := range []string{constants.DriverMariaDB, ""} {
		if got := strings.Join(domain.CapabilitiesFor(declared, server).Notes, "\n"); strings.Contains(got, "declared ") {
			t.Fatalf("declared %q was reported as a mismatch: %s", declared, got)
		}
	}
}
