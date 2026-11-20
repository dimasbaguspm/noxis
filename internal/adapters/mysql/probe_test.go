package mysql

import (
	"noxis/internal/domain"
	"strings"
	"testing"
)

func TestDSNCarriesTheTarget(t *testing.T) {
	target := domain.Target{
		Host: "db.internal", Port: 3307, Database: "app", Username: "noxis",
		TLSMode: "skip-verify", Options: map[string]string{"charset": "utf8mb4"},
	}
	dsn := dsnOf(target, "s3cret")
	for _, want := range []string{
		"noxis:s3cret@tcp(db.internal:3307)/app",
		"parseTime=true",
		"tls=skip-verify",
		"charset=utf8mb4",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn is missing %q: %s", want, dsn)
		}
	}
}

func TestTLSModeMapsOneToOne(t *testing.T) {
	for mode, want := range map[string]string{
		"preferred":   "tls=preferred",
		"skip-verify": "tls=skip-verify",
		"verify":      "tls=true",
		"":            "tls=preferred",
	} {
		dsn := dsnOf(domain.Target{Host: "h", Port: 3306, TLSMode: mode}, "")
		if !strings.Contains(dsn, want) {
			t.Fatalf("mode %q produced %s, want %s", mode, dsn, want)
		}
	}
}

func TestParseServerVersion(t *testing.T) {
	for _, tc := range []struct {
		name                string
		version, comment    string
		flavor              string
		major, minor, patch int
	}{
		{name: "mysql 8", version: "8.0.36", comment: "MySQL Community Server - GPL", flavor: "mysql", major: 8, minor: 0, patch: 36},
		{name: "mysql 8.4", version: "8.4.2", comment: "MySQL Community Server - GPL", flavor: "mysql", major: 8, minor: 4, patch: 2},
		{name: "mariadb 10.11", version: "10.11.6-MariaDB-1:10.11.6+maria~ubu2204", comment: "mariadb.org binary distribution", flavor: "mariadb", major: 10, minor: 11, patch: 6},
		{name: "mariadb 11", version: "11.4.2-MariaDB", comment: "mariadb.org binary distribution", flavor: "mariadb", major: 11, minor: 4, patch: 2},
		{name: "mariadb named only in the comment", version: "5.5.5-10.6.16", comment: "MariaDB Server", flavor: "mariadb", major: 5, minor: 5, patch: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.ParseServerVersion(tc.version, tc.comment)
			if got.Flavor != tc.flavor || got.Major != tc.major || got.Minor != tc.minor || got.Patch != tc.patch {
				t.Fatalf("got %+v, want flavor %s %d.%d.%d", got, tc.flavor, tc.major, tc.minor, tc.patch)
			}
		})
	}
}
