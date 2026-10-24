package pgmock

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestServesSimpleQuery(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Answer([]string{"n"}, []string{"42"})

	db, err := sql.Open("pgx", fmt.Sprintf("postgres://mock:mock@127.0.0.1:%d/demo?sslmode=disable&default_query_exec_mode=simple_protocol", server.Port()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	var n string
	if err := db.QueryRow("SELECT 42 AS n").Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != "42" {
		t.Fatalf("n = %q, want 42", n)
	}
	queries := server.Queries()
	if len(queries) == 0 || queries[len(queries)-1] != "SELECT 42 AS n" {
		t.Fatalf("ledger = %v", queries)
	}
}

func TestServesError(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Fail("relation missing does not exist")

	db, err := sql.Open("pgx", fmt.Sprintf("postgres://mock:mock@127.0.0.1:%d/demo?sslmode=disable&default_query_exec_mode=simple_protocol", server.Port()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Query("SELECT * FROM missing"); err == nil {
		t.Fatal("expected a target error")
	}
}
