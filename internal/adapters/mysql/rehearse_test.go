package mysql

import (
	"errors"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// A target that may not create a database is a target a static check can still be honest about. Every
// other refusal is a failure, and a wrong password is not a degradation at all: nothing connected, so
// nothing can be claimed about the change.
func TestRefusedForPrivilege(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"access denied to the database", &mysqldriver.MySQLError{Number: 1044, Message: "Access denied"}, true},
		{"create command denied", &mysqldriver.MySQLError{Number: 1142, Message: "command denied"}, true},
		{"wrong password", &mysqldriver.MySQLError{Number: 1045, Message: "Access denied"}, false},
		{"bad statement", &mysqldriver.MySQLError{Number: 1064, Message: "syntax"}, false},
		{"not a driver error", errors.New("connection reset"), false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := refusedForPrivilege(test.err); got != test.want {
				t.Fatalf("refusedForPrivilege(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

// A degraded rehearsal has to say three things in one note: what failed, that the proof is the weaker
// one, and the grant that would restore the real one.
func TestStaticInsteadNoteSaysWhatItIsAndWhy(t *testing.T) {
	note := staticInsteadNote("no privilege: Error 1044 (42000): Access denied")
	for _, want := range []string{"Error 1044", "only a static check", "grant CREATE DATABASE"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note = %q, want it to mention %q", note, want)
		}
	}
}
