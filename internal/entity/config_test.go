package entity

import "testing"

// Guards the env contract: LoadConfig must wire every env var into the
// struct and fieldOf must resolve it for Require (both broke once, silently).
func TestLoadConfigRequire(t *testing.T) {
	for _, e := range []string{EnvListen, EnvDB} {
		t.Setenv(e, "x")
	}
	if err := (LoadConfig()).Require(EnvListen, EnvDB); err != nil {
		t.Fatalf("wired envs must pass: %v", err)
	}
	t.Setenv(EnvDB, "")
	if err := (LoadConfig()).Require(EnvListen, EnvDB); err == nil {
		t.Fatal("missing env must fail Require")
	}
	if err := (LoadConfig()).Require("NOXIS_DOES_NOT_EXIST"); err == nil {
		t.Fatal("unknown env must fail Require")
	}
}
