package usecases_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"noxis/internal/entity"
	"noxis/internal/mocks"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

// TestRequireSuper needs no dependency calls: the gate reads the principal only.
func TestRequireSuper(t *testing.T) {
	deps := usecases.RegistryDeps{}
	if err := deps.RequireSuper(context.Background()); err == nil {
		t.Fatal("no principal must be denied")
	}
	member := entity.WithPrincipal(context.Background(), entity.Identity{Subject: "alice"})
	if err := deps.RequireSuper(member); err == nil {
		t.Fatal("a member must be denied")
	}
	root := entity.WithPrincipal(context.Background(), entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if err := deps.RequireSuper(root); err != nil {
		t.Fatalf("the super admin must pass: %v", err)
	}
}

func TestAuthorizeIgnoresExpiredGrant(t *testing.T) {
	perms := mocks.NewMockPermissionRepository(gomock.NewController(t))
	deps := usecases.RegistryDeps{Registry: repositories.Registry{Permissions: perms}}
	ctx := entity.WithPrincipal(context.Background(), entity.Identity{Subject: "local:alice"})
	grant := func(expiresAt int64) []entity.Permission {
		return []entity.Permission{{
			Connection: "demo", Member: "local:alice", MemberType: entity.MemberUser,
			Level: entity.LevelWrite, ExpiresAt: expiresAt,
		}}
	}

	perms.EXPECT().ListPermissions(gomock.Any(), "demo").Return(grant(time.Now().Add(-time.Hour).Unix()), nil)
	if err := deps.Authorize(ctx, "demo", entity.LevelRead); err == nil {
		t.Fatal("an expired grant must not authorize")
	}

	perms.EXPECT().ListPermissions(gomock.Any(), "demo").Return(grant(time.Now().Add(time.Hour).Unix()), nil)
	if err := deps.Authorize(ctx, "demo", entity.LevelWrite); err != nil {
		t.Fatalf("a live grant must authorize: %v", err)
	}
}
