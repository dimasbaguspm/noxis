package usecases_test

import (
	"context"
	"testing"
	"time"

	"noxis/internal/entity"
	"noxis/internal/usecases"
	"noxis/internal/usecases/ucstest"
)

func TestRequireSuper(t *testing.T) {
	d := ucstest.New(nil).Deps()
	if err := d.RequireSuper(context.Background()); err == nil {
		t.Fatal("no principal must be denied")
	}
	member := usecases.WithPrincipal(context.Background(), entity.Identity{Subject: "alice"})
	if err := d.RequireSuper(member); err == nil {
		t.Fatal("a member must be denied")
	}
	root := usecases.WithPrincipal(context.Background(), entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if err := d.RequireSuper(root); err != nil {
		t.Fatalf("the super admin must pass: %v", err)
	}
}

func TestAuthorizeIgnoresExpiredGrant(t *testing.T) {
	f := ucstest.New(nil)
	d := f.Deps()
	ctx := usecases.WithPrincipal(context.Background(), entity.Identity{Subject: "local:alice"})
	f.Perms.M["local:alice"] = entity.Permission{
		Connection: "demo", Member: "local:alice", MemberType: entity.MemberUser,
		Level: entity.LevelWrite, ExpiresAt: time.Now().Add(-time.Hour).Unix(),
	}
	if err := d.Authorize(ctx, "demo", entity.LevelRead); err == nil {
		t.Fatal("an expired grant must not authorize")
	}

	f.Perms.M["local:alice"] = entity.Permission{
		Connection: "demo", Member: "local:alice", MemberType: entity.MemberUser,
		Level: entity.LevelWrite, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	if err := d.Authorize(ctx, "demo", entity.LevelWrite); err != nil {
		t.Fatalf("a live grant must authorize: %v", err)
	}
}
