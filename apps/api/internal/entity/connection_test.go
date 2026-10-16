package entity

import "testing"

// Validation is per driver: SQL needs a user, Redis usually has none.
func TestConnectionValidation(t *testing.T) {
	base := Connection{Name: "c", Host: "h", Port: 1, Database: "d"}
	sql := base
	sql.Driver, sql.User = "postgres", ""
	if err := sql.Validate(); err == nil {
		t.Fatal("postgres without a user must fail validation")
	}
	redis := base
	redis.Driver, redis.User = "redis", ""
	if err := redis.Validate(); err != nil {
		t.Fatalf("redis without a user must pass: %v", err)
	}
	if err := (Connection{Name: "c", Driver: "redis", Host: "h", Port: 6379}).Validate(); err == nil {
		t.Fatal("redis without a database index must fail validation")
	}
	valkey := base
	valkey.Driver, valkey.User = "valkey", ""
	if err := valkey.Validate(); err != nil {
		t.Fatalf("valkey without a user must pass: %v", err)
	}

	// SQLite is file-backed: only the path (Database) is required.
	if err := (Connection{Name: "c", Driver: "sqlite", Database: "/data/app.db"}).Validate(); err != nil {
		t.Fatalf("sqlite with a path must pass: %v", err)
	}
	if err := (Connection{Name: "c", Driver: "sqlite"}).Validate(); err == nil {
		t.Fatal("sqlite without a path must fail validation")
	}
	noPort := base
	noPort.Driver, noPort.User, noPort.Port = "postgres", "u", 0
	if err := noPort.Validate(); err == nil {
		t.Fatal("postgres without a port must fail validation")
	}
}
