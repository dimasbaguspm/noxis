package permission

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/mocks"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/common/testkit"
	"noxis/internal/usecases"
)

// harness keeps one connection and its grants behind the repository mocks.
type harness struct {
	deps   usecases.RegistryDeps
	grants map[string]entity.Permission
	users  []entity.User
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &harness{grants: map[string]entity.Permission{}}
	conns := mocks.NewMockConnectionRepository(ctrl)
	perms := mocks.NewMockPermissionRepository(ctrl)
	users := mocks.NewMockUserRepository(ctrl)
	sq := mocks.NewMockSQLModule(ctrl)

	conns.EXPECT().Get(gomock.Any(), "demo").Return(&entity.Connection{
		Name: "demo", Driver: constants.DriverPostgres, Family: constants.FamilySQL,
	}, nil).AnyTimes()
	sq.EXPECT().Family().Return(constants.FamilySQL).AnyTimes()
	sq.EXPECT().Supports(gomock.Any()).Return(true).AnyTimes()

	perms.EXPECT().ListPermissions(gomock.Any(), "demo").DoAndReturn(func(context.Context, string) ([]entity.Permission, error) {
		out := make([]entity.Permission, 0, len(h.grants))
		for _, p := range h.grants {
			out = append(out, p)
		}
		return out, nil
	}).AnyTimes()
	perms.EXPECT().SetPermission(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p entity.Permission) error {
		h.grants[p.Member] = p
		return nil
	}).AnyTimes()
	perms.EXPECT().DeletePermission(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _, member, _ string) error {
			delete(h.grants, member)
			return nil
		}).AnyTimes()
	perms.EXPECT().ListGroupNames(gomock.Any()).DoAndReturn(func(context.Context) ([]string, error) {
		seen := map[string]struct{}{}
		out := []string{}
		for _, p := range h.grants {
			if p.MemberType != entity.MemberGroup {
				continue
			}
			if _, ok := seen[p.Member]; ok {
				continue
			}
			seen[p.Member] = struct{}{}
			out = append(out, p.Member)
		}
		return out, nil
	}).AnyTimes()
	users.EXPECT().ListUsers(gomock.Any()).DoAndReturn(func(context.Context) ([]entity.User, error) {
		return h.users, nil
	}).AnyTimes()

	h.deps = usecases.RegistryDeps{
		Registry: repositories.Registry{Connections: conns, Permissions: perms, Users: users},
		Modules:  modules.New(map[string]interfaces.Module{constants.FamilySQL: sq}, nil),
	}
	return h
}

func TestListRequiresInstanceAdmin(t *testing.T) {
	h := newHarness(t)
	mux := testkit.Mux("GET", "/external/v1/engine/{engine}/connections/{name}/permissions", List{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no principal: %d %s", rec.Code, rec.Body.String())
	}
	// a member with no grant on the connection is forbidden
	rec = testkit.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", &entity.Identity{Subject: "local:bob"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin list: %d %s", rec.Code, rec.Body.String())
	}
}

func TestCRUD(t *testing.T) {
	h := newHarness(t)
	admin := &entity.Identity{Subject: "root", Role: entity.RoleSuper}

	put := testkit.Mux("PUT", "/external/v1/engine/{engine}/connections/{name}/permissions", Set{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(put, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"write"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status=%d body=%s", rec.Code, rec.Body.String())
	}

	get := testkit.Mux("GET", "/external/v1/engine/{engine}/connections/{name}/permissions", List{RegistryDeps: h.deps}.Serve)
	rec = testkit.Do(get, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", admin)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"member":"dev"`) {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = testkit.Do(put, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"root"}`, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad level status=%d want 400", rec.Code)
	}

	del := testkit.Mux("DELETE", "/external/v1/engine/{engine}/connections/{name}/permissions", Delete{RegistryDeps: h.deps}.Serve)
	rec = testkit.Do(del, http.MethodDelete, "/external/v1/engine/sql/connections/demo/permissions?member=dev", "", admin)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMembers(t *testing.T) {
	h := newHarness(t)
	h.grants["readers"] = entity.Permission{Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead}
	h.users = []entity.User{{Subject: "local:alice", Username: "alice"}}

	rec := testkit.Do(testkit.Mux("GET", "/external/v1/config/members", Members{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/external/v1/config/members", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"readers"`) || !strings.Contains(rec.Body.String(), `"local:alice"`) {
		t.Fatalf("members status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInstanceAdminCanReadAndRevokeButNotGrant(t *testing.T) {
	h := newHarness(t)
	h.grants["local:bob"] = entity.Permission{
		Connection: "demo", Member: "local:bob", MemberType: entity.MemberUser,
		Level: entity.LevelAdmin, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	h.grants["dev"] = entity.Permission{
		Connection: "demo", Member: "dev", MemberType: entity.MemberGroup,
		Level: entity.LevelRead, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	bob := &entity.Identity{Subject: "local:bob"}

	rec := testkit.Do(testkit.Mux("GET", "/x/{engine}/c/{name}", List{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/x/sql/c/demo", "", bob)
	if rec.Code != http.StatusOK {
		t.Fatalf("instance admin list: %d %s", rec.Code, rec.Body.String())
	}

	rec = testkit.Do(testkit.Mux("PUT", "/x/{engine}/c/{name}", Set{RegistryDeps: h.deps}.Serve),
		http.MethodPut, "/x/sql/c/demo", `{"member":"x","memberType":"group","level":"read"}`, bob)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("instance admin must not grant: %d %s", rec.Code, rec.Body.String())
	}

	// the instance's own admin may take access away
	rec = testkit.Do(testkit.Mux("DELETE", "/x/{engine}/c/{name}", Delete{RegistryDeps: h.deps}.Serve),
		http.MethodDelete, "/x/sql/c/demo?member=dev", "", bob)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("instance admin revoke: %d %s", rec.Code, rec.Body.String())
	}

	// a member with no admin grant on the connection may not
	carol := &entity.Identity{Subject: "local:carol"}
	rec = testkit.Do(testkit.Mux("DELETE", "/x/{engine}/c/{name}", Delete{RegistryDeps: h.deps}.Serve),
		http.MethodDelete, "/x/sql/c/demo?member=dev", "", carol)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin must not revoke: %d %s", rec.Code, rec.Body.String())
	}
}

// A promoted admin manages the grant policy of every connection without holding
// a grant on it; a member still cannot.
func TestGlobalAdminManagesEveryConnection(t *testing.T) {
	h := newHarness(t)
	admin := &entity.Identity{Subject: "local:admin", Role: entity.RoleAdmin}
	member := &entity.Identity{Subject: "local:bob", Role: entity.RoleMember}

	rec := testkit.Do(testkit.Mux("PUT", "/x/{engine}/c/{name}", Set{RegistryDeps: h.deps}.Serve),
		http.MethodPut, "/x/sql/c/demo", `{"member":"dev","memberType":"group","level":"write"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("global admin grant: %d %s", rec.Code, rec.Body.String())
	}
	rec = testkit.Do(testkit.Mux("PUT", "/x/{engine}/c/{name}", Set{RegistryDeps: h.deps}.Serve),
		http.MethodPut, "/x/sql/c/demo", `{"member":"dev","memberType":"group","level":"write"}`, member)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member grant must be forbidden: %d %s", rec.Code, rec.Body.String())
	}
	rec = testkit.Do(testkit.Mux("GET", "/x/{engine}/c/{name}", List{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/x/sql/c/demo", "", admin)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"dev"`) {
		t.Fatalf("global admin list: %d %s", rec.Code, rec.Body.String())
	}
}
