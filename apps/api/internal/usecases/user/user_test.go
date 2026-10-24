package user_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"noxis/internal/common/testkit"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/mocks"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
	"noxis/internal/usecases/user"
)

// harness keeps the account rows in one slice and serves them through the repository mocks, so a test asserts on the stored state after the request.
type harness struct {
	deps   usecases.RegistryDeps
	users  []entity.User
	grants []entity.Permission
	events []entity.AccessEvent
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &harness{
		users: []entity.User{
			{Subject: "local:root@example.com", Email: "root@example.com", Username: "root@example.com", Role: entity.RoleSuper, Status: entity.StatusActive},
			{Subject: "local:admin@example.com", Email: "admin@example.com", Username: "admin@example.com", Role: entity.RoleAdmin, Status: entity.StatusActive},
			{Subject: "local:bob@example.com", Email: "bob@example.com", Username: "bob@example.com", Groups: []string{"devs"}, Role: entity.RoleMember, Status: entity.StatusActive},
			{Subject: "oidc:carol", Email: "carol@example.com", Role: entity.RoleMember, Status: entity.StatusBanned},
		},
		grants: []entity.Permission{{
			Connection: "demo", Member: "local:bob@example.com", MemberType: entity.MemberUser, Level: entity.LevelWrite,
		}},
	}
	users := mocks.NewMockUserRepository(ctrl)
	perms := mocks.NewMockPermissionRepository(ctrl)
	events := mocks.NewMockAccessEventRepository(ctrl)

	users.EXPECT().ListUsers(gomock.Any()).DoAndReturn(func(context.Context) ([]entity.User, error) {
		return h.users, nil
	}).AnyTimes()
	users.EXPECT().GetUser(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, subject string) (*entity.User, error) {
		for i := range h.users {
			if h.users[i].Subject == subject {
				u := h.users[i]
				return &u, nil
			}
		}
		return nil, constants.ErrNotFound
	}).AnyTimes()
	users.EXPECT().SetUserStatus(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, subject, status string) error {
			for i := range h.users {
				if h.users[i].Subject == subject {
					h.users[i].Status = status
					return nil
				}
			}
			return constants.ErrNotFound
		}).AnyTimes()
	users.EXPECT().SetUserRole(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, subject, role string) error {
			for i := range h.users {
				if h.users[i].Subject == subject {
					h.users[i].Role = role
					return nil
				}
			}
			return constants.ErrNotFound
		}).AnyTimes()

	perms.EXPECT().ListPermissionsForMembers(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, members []string) ([]entity.Permission, error) {
			want := map[string]struct{}{}
			for _, m := range members {
				want[m] = struct{}{}
			}
			out := []entity.Permission{}
			for i := range h.grants {
				p := &h.grants[i]
				if _, ok := want[p.Member]; ok {
					out = append(out, *p)
				}
			}
			return out, nil
		}).AnyTimes()
	events.EXPECT().ListAccessEvents(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, subject string, limit int) ([]entity.AccessEvent, error) {
			out := []entity.AccessEvent{}
			for i := range h.events {
				e := &h.events[i]
				if e.Subject == subject || e.Actor == subject {
					out = append(out, *e)
				}
			}
			if limit > 0 && len(out) > limit {
				out = out[:limit]
			}
			return out, nil
		}).AnyTimes()

	h.deps = usecases.RegistryDeps{Repositories: repositories.Registry{
		Users: users, Permissions: perms, AccessEvents: events,
	}}
	return h
}

// roleOf is the stored role of one account.
func (h *harness) roleOf(subject string) string {
	for i := range h.users {
		if h.users[i].Subject == subject {
			return h.users[i].Role
		}
	}
	return ""
}

// statusOf is the stored status of one account.
func (h *harness) statusOf(subject string) string {
	for i := range h.users {
		if h.users[i].Subject == subject {
			return h.users[i].Status
		}
	}
	return ""
}

func identities() (super, admin *entity.Identity) {
	return &entity.Identity{Subject: "local:root@example.com", Role: entity.RoleSuper},
		&entity.Identity{Subject: "local:admin@example.com", Role: entity.RoleAdmin}
}

func TestDirectoryFiltersAndGuard(t *testing.T) {
	h := newHarness(t)
	super, admin := identities()
	mux := testkit.Mux("GET", "/users", user.List{RegistryDeps: h.deps}.Serve)

	for name, caller := range map[string]*entity.Identity{"super": super, "admin": admin} {
		rec := testkit.Do(mux, http.MethodGet, "/users", "", caller)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "local:bob@example.com") {
			t.Fatalf("%s list: %d %s", name, rec.Code, rec.Body.String())
		}
	}

	rec := testkit.Do(mux, http.MethodGet, "/users?q=bob@example.com", "", super)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "local:root@example.com") {
		t.Fatalf("search: %d %s", rec.Code, rec.Body.String())
	}

	rec = testkit.Do(mux, http.MethodGet, "/users?status=banned", "", super)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "oidc:carol") || strings.Contains(rec.Body.String(), "local:bob@example.com") {
		t.Fatalf("status filter: %d %s", rec.Code, rec.Body.String())
	}

	rec = testkit.Do(mux, http.MethodGet, "/users", "", &entity.Identity{Subject: "local:bob@example.com", Role: entity.RoleMember})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a member must be refused: %d", rec.Code)
	}
}

func TestDetailCarriesAccessAndEvents(t *testing.T) {
	h := newHarness(t)
	_, admin := identities()
	rec := testkit.Do(testkit.Mux("GET", "/users/{subject}", user.Get{RegistryDeps: h.deps}.Serve),
		http.MethodGet, "/users/local:bob@example.com", "", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"access"`) || !strings.Contains(body, `"demo"`) || !strings.Contains(body, `"events"`) {
		t.Fatalf("detail body: %s", body)
	}
}

func TestStatusGuards(t *testing.T) {
	h := newHarness(t)
	super, admin := identities()
	mux := testkit.Mux("PATCH", "/users/{subject}", user.Update{RegistryDeps: h.deps}.Serve)

	rec := testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"status":"banned"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ban: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.statusOf("local:bob@example.com"); got == entity.StatusActive {
		t.Fatal("bob must be banned")
	}

	rec = testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"status":"active"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin unban: %d %s", rec.Code, rec.Body.String())
	}

	// the root is never bannable, and nobody edits themselves
	for name, tc := range map[string]struct {
		subject string
		caller  *entity.Identity
	}{
		"super target": {"local:root@example.com", super},
		"self":         {"local:admin@example.com", admin},
	} {
		rec := testkit.Do(mux, http.MethodPatch, "/users/"+tc.subject, `{"status":"banned"}`, tc.caller)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s must be forbidden: %d", name, rec.Code)
		}
	}

	// an admin may not touch a peer admin
	h.users = append(h.users, entity.User{
		Subject: "local:other@example.com", Email: "other@example.com", Role: entity.RoleAdmin, Status: entity.StatusActive,
	})
	rec = testkit.Do(mux, http.MethodPatch, "/users/local:other@example.com", `{"status":"banned"}`, admin)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin banning a peer must be forbidden: %d", rec.Code)
	}
	rec = testkit.Do(mux, http.MethodPatch, "/users/local:other@example.com", `{"status":"banned"}`, super)
	if rec.Code != http.StatusOK {
		t.Fatalf("super banning an admin: %d %s", rec.Code, rec.Body.String())
	}

	if rec := testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{}`, super); rec.Code != http.StatusBadRequest {
		t.Fatalf("an empty patch must be a 400: %d", rec.Code)
	}
	if rec := testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"status":"root"}`, super); rec.Code != http.StatusBadRequest {
		t.Fatalf("an unknown status must be a 400: %d", rec.Code)
	}
}

// a patch whose second field is invalid must not apply its first field
func TestPatchValidatesBeforeMutating(t *testing.T) {
	h := newHarness(t)
	super, _ := identities()
	mux := testkit.Mux("PATCH", "/users/{subject}", user.Update{RegistryDeps: h.deps}.Serve)

	rec := testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"admin","status":"root"}`, super)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid patch must be a 400: %d", rec.Code)
	}
	if got := h.roleOf("local:bob@example.com"); got == entity.RoleAdmin {
		t.Fatal("the role must not be applied when the status is invalid")
	}
}

func TestRoleIsSuperOnly(t *testing.T) {
	h := newHarness(t)
	super, admin := identities()
	mux := testkit.Mux("PATCH", "/users/{subject}", user.Update{RegistryDeps: h.deps}.Serve)

	rec := testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"admin"}`, admin)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("an admin must not promote: %d %s", rec.Code, rec.Body.String())
	}

	rec = testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"admin"}`, super)
	if rec.Code != http.StatusOK {
		t.Fatalf("super promote: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.roleOf("local:bob@example.com"); got != entity.RoleAdmin {
		t.Fatalf("bob role = %q, want admin", got)
	}

	rec = testkit.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"super"}`, super)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a second super must be a 400: %d", rec.Code)
	}
}
