package mysqlmock

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func openMock(t *testing.T, server *Server) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", fmt.Sprintf("mock:mock@tcp(127.0.0.1:%d)/demo", server.Port()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestServesQueryAndPing(t *testing.T) {
	server, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	server.Answer([]string{"n"}, []string{"42"})
	db := openMock(t, server)

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
	server.Fail("table missing does not exist")
	db := openMock(t, server)

	if _, err := db.Query("SELECT * FROM missing"); err == nil {
		t.Fatal("expected a target error")
	}
}
