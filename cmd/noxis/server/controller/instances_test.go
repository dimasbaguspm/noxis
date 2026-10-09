package controller

import (
	"context"
	"errors"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The cross-family lookup must decorate an instance with its family, never
// leak the password, and 404 an id that does not exist.
func TestInstanceLookup(t *testing.T) {
	d, changes, conns := newDeps(&fakeSQL{})
	ctx := context.Background()

	c := conns.conns["demo"]
	c.ID = "abc"
	c.Password = "secret"
	conns.conns["demo"] = c
	if err := changes.SaveChange(ctx, entity.Change{
		ID: "c1", Connection: "demo", Kind: constants.KindDDL, Status: constants.ChangeApplied,
	}); err != nil {
		t.Fatalf("seed change: %v", err)
	}
	if err := changes.SaveChange(ctx, entity.Change{
		ID: "c2", Connection: "other", Kind: constants.KindDDL, Status: constants.ChangeApplied,
	}); err != nil {
		t.Fatalf("seed other change: %v", err)
	}

	got, err := d.instance(ctx, "abc")
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	if got.Name != "demo" || got.Family != constants.FamilySQL {
		t.Fatalf("instance = %+v", got)
	}
	if got.Password != "" {
		t.Fatal("password must be redacted")
	}

	if _, err := d.instance(ctx, "missing"); !errors.Is(err, interfaces.ErrNotFound) {
		t.Fatalf("missing id: err=%v want not found", err)
	}

	list, err := d.instanceChanges(ctx, "abc")
	if err != nil {
		t.Fatalf("instanceChanges: %v", err)
	}
 if len(list) != 1 || list[0].ID != "c1" {
		t.Fatalf("changes = %+v, want only c1", list)
	}
}

// fakeMemcache is just enough of a module for the cross-family list: it names
// its family and claims the memory-cache drivers.
type fakeMemcache struct{ interfaces.Module }

func (fakeMemcache) Family() string { return constants.FamilyMemcache }

func (fakeMemcache) Supports(driver string) bool {
	return driver == constants.DriverRedis || driver == constants.DriverValkey
}

// The list is cross-family and its filters are name/host substrings plus an
// exact driver match.
func TestInstanceSearch(t *testing.T) {
	d, _, conns := newDeps(&fakeSQL{})
	ctx := context.Background()
	conns.conns["prod-db"] = entity.Connection{
		Name: "prod-db", Driver: constants.DriverPostgres, Host: "db.internal", Port: 5432,
		User: "postgres", Database: "app",
	}
	conns.conns["cache"] = entity.Connection{
		Name: "cache", Driver: constants.DriverRedis, Host: "redis.internal", Port: 6379, Database: "0",
	}
	memcache := &fakeMemcache{}
	d.Modules[constants.FamilyMemcache] = memcache

	all, err := d.instances(ctx, "", "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("all = %d, want 3", len(all))
	}

	byName, err := d.instances(ctx, "PROD", "", "")
	if err != nil {
		t.Fatalf("by name: %v", err)
	}
	if len(byName) != 1 || byName[0].Name != "prod-db" {
		t.Fatalf("by name = %+v", byName)
	}

	byDriver, err := d.instances(ctx, "", constants.DriverRedis, "")
	if err != nil {
		t.Fatalf("by driver: %v", err)
	}
	if len(byDriver) != 1 || byDriver[0].Family != constants.FamilyMemcache {
		t.Fatalf("by driver = %+v", byDriver)
	}

	byHost, err := d.instances(ctx, "", "", "internal")
	if err != nil {
		t.Fatalf("by host: %v", err)
	}
	if len(byHost) != 2 {
		t.Fatalf("by host = %+v, want 2", byHost)
	}
}

// Identity is fixed after registration; only the password rotates.
func TestIdentityLockedAndRotation(t *testing.T) {
	d, _, conns := newDeps(&fakeSQL{})
	ctx := context.Background()
	mod := d.Modules[constants.FamilySQL]

	c := conns.conns["demo"]
	c.ID = "abc"
	c.Password = "old"
	conns.conns["demo"] = c

	changed := c
	changed.Host = "elsewhere"
	if _, err := d.saveConnection(ctx, mod, changed); err == nil {
		t.Fatal("changing host must be rejected")
	}
	if conns.conns["demo"].Host != "db" {
		t.Fatal("stored instance must be unchanged")
	}

	if err := d.rotatePassword(ctx, "abc", "new"); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if conns.conns["demo"].Password != "new" {
		t.Fatal("password not rotated")
	}
	if err := d.rotatePassword(ctx, "abc", ""); err == nil {
		t.Fatal("empty password must be rejected")
	}
	if err := d.rotatePassword(ctx, "missing", "x"); !errors.Is(err, interfaces.ErrNotFound) {
		t.Fatalf("missing id: err=%v want not found", err)
	}
}
