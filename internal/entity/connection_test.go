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
}
