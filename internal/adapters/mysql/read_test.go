package mysql

import (
	"errors"
	"noxis/internal/domain"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// Every identifier that reaches the engine from a path is quoted here, and a backtick inside a name
// is the one character that can end the quoting early. Doubling it is what keeps a table named
//
//	users` ; DROP DATABASE app ; --
//
// a table and not two statements.
func TestQuoteDoublesBackticks(t *testing.T) {
	for _, tc := range []struct {
		name, identifier, want string
	}{
		{"a plain name", "users", "`users`"},
		{"an empty name", "", "``"},
		{"a name with a backtick", "we`ird", "`we``ird`"},
		{"a name that is only backticks", "``", "``````"},
		{"a name with a space", "user table", "`user table`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := quote(tc.identifier); got != tc.want {
				t.Fatalf("quote(%q) = %q, want %q", tc.identifier, got, tc.want)
			}
		})
	}
}

// A NULL in a result is a fact a reader has to see, so it stays nil; everything else is rendered as
// text, and a timestamp is rendered as UTC so two servers in two zones do not print two answers.
func TestStringsOfRendersEachScanType(t *testing.T) {
	stamp := time.Date(2024, 5, 6, 7, 8, 9, 123456789, time.FixedZone("CEST", 2*60*60))
	row := []any{nil, []byte("bytes"), "text", stamp, 42, int64(7), true, 1.5}
	got := stringsOf(row)
	want := []*string{
		nil,
		text("bytes"),
		text("text"),
		text("2024-05-06T05:08:09.123456789Z"),
		text("42"),
		text("7"),
		text("true"),
		text("1.5"),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for i := range want {
		switch {
		case want[i] == nil && got[i] != nil:
			t.Errorf("column %d = %q, want nil for a NULL", i, *got[i])
		case want[i] != nil && got[i] == nil:
			t.Errorf("column %d = nil, want %q", i, *want[i])
		case want[i] != nil && *want[i] != *got[i]:
			t.Errorf("column %d = %q, want %q", i, *got[i], *want[i])
		}
	}
}

// A result with no columns is not an error and not a crash: the columns are what the engine sent,
// and a query that sends none sends none.
func TestStringsOfEmptyRow(t *testing.T) {
	if got := stringsOf(nil); len(got) != 0 {
		t.Fatalf("stringsOf(nil) = %v, want an empty row", got)
	}
}

// What the engine says has to come back as what the caller should do: a missing schema is a 404, a
// typo is a 400, a refusal is a 403, and anything else is the engine being unsupported rather than
// the request being wrong.
func TestStatementErrorMapsTheEngineNumber(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"unknown database", &mysqldriver.MySQLError{Number: 1049, Message: "Unknown database"}, domain.ErrNotFound},
		{"unknown table", &mysqldriver.MySQLError{Number: 1146, Message: "Table doesn't exist"}, domain.ErrNotFound},
		{"unknown column", &mysqldriver.MySQLError{Number: 1054, Message: "Unknown column"}, domain.ErrInvalid},
		{"syntax error", &mysqldriver.MySQLError{Number: 1064, Message: "You have an error in your SQL syntax"}, domain.ErrInvalid},
		{"access denied to the schema", &mysqldriver.MySQLError{Number: 1044, Message: "Access denied"}, domain.ErrForbidden},
		{"wrong password", &mysqldriver.MySQLError{Number: 1045, Message: "Access denied"}, domain.ErrUnsupported},
		{"not the engine's error", errors.New("connection reset"), domain.ErrUnsupported},
		{"wrapped engine error", errors.Join(errors.New("while reading"), &mysqldriver.MySQLError{Number: 1146}), domain.ErrNotFound},
		{"no error", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := statementError(tc.err, "")
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("statementError(nil) = %v, want nil", got)
			case tc.want == nil:
				return
			case !errors.Is(got, tc.want):
				t.Fatalf("statementError(%v) = %v, want it to wrap %v", tc.err, got, tc.want)
			}
			if tc.err != nil && !strings.Contains(got.Error(), tc.err.Error()) {
				t.Fatalf("statementError(%v) = %v, want the engine's own words kept", tc.err, got)
			}
		})
	}
}

// A password must never travel back out in an error: a driver that echoes the DSN back, or a
// server that quotes what it was given, is the one path by which it could.
func TestStatementErrorScrubsThePassword(t *testing.T) {
	err := &mysqldriver.MySQLError{Number: 1045, Message: "Access denied for user 'noxis'@'10.0.0.1' (using password: s3cret)"}
	got := statementError(err, "s3cret")
	if strings.Contains(got.Error(), "s3cret") {
		t.Fatalf("the password survived into %v", got)
	}
	if !strings.Contains(got.Error(), "***") {
		t.Fatalf("the password was not replaced by a marker: %v", got)
	}
}

// scrub is the single place that cut happens, so it is tested on its own: with no password there is
// nothing to replace, and the message is passed through untouched.
func TestScrub(t *testing.T) {
	for _, tc := range []struct {
		name, message, password, want string
	}{
		{"a password in the message", "dial noxis:s3cret@tcp(db:3306)/app", "s3cret", "dial noxis:***@tcp(db:3306)/app"},
		{"occurrences twice", "s3cret s3cret", "s3cret", "*** ***"},
		{"no password configured", "dial db:3306: refused", "", "dial db:3306: refused"},
		{"a password that is not there", "dial db:3306: refused", "s3cret", "dial db:3306: refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := scrub(errors.New(tc.message), tc.password); got != tc.want {
				t.Fatalf("scrub = %q, want %q", got, tc.want)
			}
		})
	}
}

// connectError is what a caller sees when the probe could not say who the server is; it must carry
// ErrUnsupported so the transport has one status to map, and it must not carry the password.
func TestConnectErrorScrubsAndWraps(t *testing.T) {
	got := connectError(errors.New("dial error with s3cret in it"), "s3cret")
	if !errors.Is(got, domain.ErrUnsupported) {
		t.Fatalf("connectError = %v, want it to wrap ErrUnsupported", got)
	}
	if strings.Contains(got.Error(), "s3cret") {
		t.Fatalf("the password survived into %v", got)
	}
}

// The pool is opened lazily and never dials, which is the guarantee every use case leans on: a
// target that is down costs the caller nothing until a statement is actually sent.
func TestPoolOpensWithoutConnecting(t *testing.T) {
	db := New(time.Second)
	pool, err := db.pool(domain.Target{Host: "127.0.0.1", Port: 3306, Database: "app", Username: "noxis"}, "")
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer func() { _ = pool.Close() }()
	if got := pool.Stats().OpenConnections; got != 0 {
		t.Fatalf("%d connections opened, want none until a statement is sent", got)
	}
}

func text(s string) *string { return &s }
