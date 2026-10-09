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
