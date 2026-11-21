package mysql

import (
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// The DSN is the one connection string this build makes, and the driver itself is the only honest
// judge of whether it parses. Every case round-trips through ParseDSN, because a string that looks
// right and is refused at connect time is the bug this file exists to prevent.
func TestDSNParsesBack(t *testing.T) {
	target := domain.Target{
		Driver: constants.DriverMySQL, Host: "db.internal", Port: 3307,
		Database: "app", Username: "noxis", TLSMode: constants.TLSSkipVerify,
		Options: map[string]string{"charset": "utf8mb4"},
	}
	dsn := dsnOf(target, "s3cret")
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("the driver refused the DSN it was given: %v", err)
	}
	for _, want := range []struct{ name, got, expect string }{
		{"user", cfg.User, "noxis"},
		{"password", cfg.Passwd, "s3cret"},
		{"address", cfg.Addr, "db.internal:3307"},
		{"database", cfg.DBName, "app"},
		{"tls", cfg.TLSConfig, "skip-verify"},
		{"charset, as the driver re-emits it", cfg.FormatDSN(), "charset=utf8mb4"},
	} {
		if !strings.Contains(want.got, want.expect) {
			t.Errorf("%s = %q, want it to carry %q (dsn: %s)", want.name, want.got, want.expect, dsn)
		}
	}
	if !cfg.ParseTime {
		t.Error("ParseTime is off: a scanned time would come back as a string")
	}
}

// Both engines are reached through one wire protocol, so the engine a target declares must not
// appear in the DSN at all: whoever assembled one by hand would otherwise have two spellings of the
// same connection to keep in step.
func TestDSNIsTheSameForBothEngines(t *testing.T) {
	base := domain.Target{Host: "h", Port: 3306, Database: "app", Username: "noxis", TLSMode: constants.TLSPreferred}
	mySQL := base
	mySQL.Driver = constants.DriverMySQL
	mariaDB := base
	mariaDB.Driver = constants.DriverMariaDB
	if got, want := dsnOf(mariaDB, "pw"), dsnOf(mySQL, "pw"); got != want {
		t.Fatalf("mariadb built %q, mysql built %q: the wire protocol is the same one", got, want)
	}
}

// A target whose account has no password is a target this build must still reach: the DSN has to
// leave the password empty rather than invent a separator, and the driver has to read it back as an
// empty password rather than as a missing one.
func TestDSNWithoutPassword(t *testing.T) {
	target := domain.Target{Host: "127.0.0.1", Port: 3306, Database: "app", Username: "noxis"}
	dsn := dsnOf(target, "")
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("the driver refused a passwordless DSN %q: %v", dsn, err)
	}
	if cfg.Passwd != "" {
		t.Fatalf("password = %q, want empty (dsn: %s)", cfg.Passwd, dsn)
	}
	if cfg.User != "noxis" {
		t.Fatalf("user = %q, want noxis (dsn: %s)", cfg.User, dsn)
	}
}

// Ports are joined, not concatenated: an IPv6 host has colons of its own, and "::1:3306" is not an
// address any server answers on.
func TestDSNJoinsTheAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		host string
		port int
		want string
	}{
		{"a hostname", "db.internal", 3306, "db.internal:3306"},
		{"an ipv4 literal", "10.0.0.7", 3307, "10.0.0.7:3307"},
		{"an ipv6 literal", "::1", 3306, "[::1]:3306"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := mysqldriver.ParseDSN(dsnOf(domain.Target{Host: tc.host, Port: tc.port}, ""))
			if err != nil {
				t.Fatalf("ParseDSN: %v", err)
			}
			if cfg.Addr != tc.want {
				t.Fatalf("addr = %q, want %q", cfg.Addr, tc.want)
			}
		})
	}
}
