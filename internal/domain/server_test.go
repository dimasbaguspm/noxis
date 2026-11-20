package domain

import (
	"strings"
	"testing"
)

func TestParseServerVersion(t *testing.T) {
	for _, tc := range []struct {
		name             string
		version, comment string
		flavor           string
		major, minor     int
	}{
		{"mysql 8.0", "8.0.36", "MySQL Community Server - GPL", "mysql", 8, 0},
		{"mysql 8.4", "8.4.2", "MySQL Community Server - GPL", "mysql", 8, 4},
		{"mariadb 10.11", "10.11.6-MariaDB-1:10.11.6+maria~ubu2204", "mariadb.org binary distribution", "mariadb", 10, 11},
		{"mariadb 11.4", "11.4.2-MariaDB", "mariadb.org binary distribution", "mariadb", 11, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseServerVersion(tc.version, tc.comment)
			if got.Flavor != tc.flavor || got.Major != tc.major || got.Minor != tc.minor {
				t.Fatalf("got %+v, want %s %d.%d", got, tc.flavor, tc.major, tc.minor)
			}
			if !got.Probed() {
				t.Fatal("a parsed version should report itself probed")
			}
		})
	}
}

func TestCapabilitiesForUnprobedTarget(t *testing.T) {
	caps := CapabilitiesFor("maria", MeasuredServer("maria", ""))
	// An unknown engine keeps the declared name, offers nothing version-dependent, and says why.
	if caps.Flavor != "maria" {
		t.Fatalf("flavor = %q", caps.Flavor)
	}
	if caps.ExplainJSON {
		t.Fatal("EXPLAIN support claimed for an engine nobody probed")
	}
	if !contains(caps.Notes, "not probed yet") {
		t.Fatalf("notes = %v", caps.Notes)
	}
}

func TestCapabilitiesForProbedTargets(t *testing.T) {
	mariadb := CapabilitiesFor("mariadb", ParseServerVersion("11.4.2-MariaDB", ""))
	if !mariadb.ExplainJSON || !mariadb.ShadowDDLRehearsal || mariadb.Flavor != "mariadb" {
		t.Fatalf("mariadb caps = %+v", mariadb)
	}
	mysql := CapabilitiesFor("mysql", ParseServerVersion("8.0.36", "MySQL Community Server - GPL"))
	if !mysql.ExplainJSON || mysql.Flavor != "mysql" {
		t.Fatalf("mysql caps = %+v", mysql)
	}
}

func TestCapabilitiesNameTheFlavorMismatch(t *testing.T) {
	caps := CapabilitiesFor("mysql", ParseServerVersion("11.4.2-MariaDB", ""))
	if caps.Flavor != "mariadb" {
		t.Fatalf("measured flavor = %q, want mariadb", caps.Flavor)
	}
	if !contains(caps.Notes, "declared mysql but the server reports mariadb") {
		t.Fatalf("notes = %v", caps.Notes)
	}
}

func contains(notes []string, want string) bool {
	for _, note := range notes {
		if strings.Contains(note, want) {
			return true
		}
	}
	return false
}
