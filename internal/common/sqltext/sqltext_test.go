package sqltext

import (
	"errors"
	"reflect"
	"testing"
)

func TestClassifyKinds(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
		kind Kind
		read bool
	}{
		{"select", "SELECT id FROM users", Select, true},
		{"select lowercase", "select 1", Select, true},
		{"explain", "EXPLAIN SELECT * FROM users", Explain, true},
		{"describe", "DESCRIBE users", Explain, true},
		{"show", "SHOW CREATE TABLE users", Show, true},
		{"insert", "INSERT INTO users (id) VALUES (1)", Insert, false},
		{"replace", "REPLACE INTO users (id) VALUES (1)", Insert, false},
		{"update", "UPDATE users SET name = 'x'", Update, false},
		{"delete", "DELETE FROM users", Delete, false},
		{"alter", "ALTER TABLE users ADD COLUMN x INT", DDL, false},
		{"drop", "DROP TABLE users", DDL, false},
		{"truncate", "TRUNCATE users", DDL, false},
		{"call", "CALL do_something()", Other, false},
		{"set", "SET @x = 1", Other, false},
		{"cte select", "WITH recent AS (SELECT 1) SELECT * FROM recent", Select, true},
		{"cte insert", "WITH recent AS (SELECT 1) INSERT INTO archive SELECT * FROM recent", Insert, false},
		{"parenthesised", "(SELECT 1)", Select, true},
		{"leading comment", "/* a note */ SELECT 1", Select, true},
		{"leading line comment", "-- a note\nSELECT 1", Select, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Classify(tc.sql)
			if err != nil {
				t.Fatalf("Classify: %v", err)
			}
			if got.Kind != tc.kind {
				t.Fatalf("kind = %q, want %q", got.Kind, tc.kind)
			}
			if got.Read() != tc.read {
				t.Fatalf("Read() = %v, want %v", got.Read(), tc.read)
			}
		})
	}
}

func TestClassifyRefusesMultiStatement(t *testing.T) {
	for _, sql := range []string{
		"SELECT 1; DROP TABLE users",
		"SELECT 1; SELECT 2",
		"SELECT 1; -- trailing\nDELETE FROM users",
	} {
		if _, err := Classify(sql); !errors.Is(err, ErrMultiple) {
			t.Fatalf("Classify(%q) = %v, want ErrMultiple", sql, err)
		}
	}
	// One trailing semicolon is still one statement.
	if _, err := Classify("SELECT 1;"); err != nil {
		t.Fatalf("Classify with a trailing semicolon: %v", err)
	}
}

func TestClassifyIgnoresKeywordsInsideCommentsAndLiterals(t *testing.T) {
	// A write hidden in a comment is a comment, and the statement is still a read.
	got, err := Classify("SELECT 'DROP TABLE users' AS note /* UPDATE users */ FROM users")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got.Kind != Select || !got.Read() {
		t.Fatalf("kind = %q, read = %v", got.Kind, got.Read())
	}
	// A comment in front of a write does not make it a read.
	got, err = Classify("/* SELECT */ DELETE FROM users")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got.Kind != Delete || got.Read() {
		t.Fatalf("kind = %q, read = %v", got.Kind, got.Read())
	}
}

func TestClassifyMarksLockingReads(t *testing.T) {
	for _, sql := range []string{
		"SELECT * FROM users FOR UPDATE",
		"SELECT * FROM users LOCK IN SHARE MODE",
		"SELECT * FROM users INTO OUTFILE '/tmp/x'",
		"SELECT * FROM users INTO DUMPFILE '/tmp/x'",
	} {
		got, err := Classify(sql)
		if err != nil {
			t.Fatalf("Classify(%q): %v", sql, err)
		}
		if !got.Locking || got.Reason == "" {
			t.Fatalf("Classify(%q) = %+v, want locking with a reason", sql, got)
		}
	}
	// A session variable is not a table.
	got, err := Classify("SELECT COUNT(*) INTO @n FROM users")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got.Locking {
		t.Fatal("a session variable counts as a locking read")
	}
}

func TestClassifyRefusesUnterminated(t *testing.T) {
	for _, sql := range []string{"SELECT 'oops", "SELECT /* oops", "SELECT `oops"} {
		if _, err := Classify(sql); err == nil {
			t.Fatalf("Classify(%q) accepted an unterminated literal or comment", sql)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	for _, sql := range []string{"", "   ", "-- only a comment\n"} {
		if _, err := Classify(sql); !errors.Is(err, ErrEmpty) && err == nil {
			t.Fatalf("Classify(%q) = %v, want an error", sql, err)
		}
	}
	if _, err := Classify("-- only a comment\n"); err == nil {
		t.Fatal("a comment-only statement classified as something")
	}
}

func TestTables(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		want []string
	}{
		{"SELECT * FROM users", []string{"users"}},
		{"SELECT * FROM app.users u JOIN orders o ON o.user_id = u.id", []string{"app.users", "orders"}},
		{"SELECT * FROM `app`.`users`", []string{"app`.`users"}},
		{"UPDATE users SET x = 1", []string{"users"}},
		{"INSERT INTO archive (id) SELECT id FROM users", []string{"archive", "users"}},
		{"SELECT 1", nil},
		{"SELECT * FROM (SELECT 1) t", nil},
	} {
		got, err := Classify(tc.sql)
		if err != nil {
			t.Fatalf("Classify(%q): %v", tc.sql, err)
		}
		if !reflect.DeepEqual(got.Tables, tc.want) {
			t.Fatalf("Classify(%q).Tables = %v, want %v", tc.sql, got.Tables, tc.want)
		}
	}
}
