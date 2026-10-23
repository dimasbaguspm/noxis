package user_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"noxis/internal/entity"
	"noxis/internal/usecases/ucstest"
	"noxis/internal/usecases/user"
)

func seed() *ucstest.Fixture {
	f := ucstest.New(&ucstest.SQL{})
	f.Users.Accounts = []entity.User{
		{Subject: "local:root@example.com", Email: "root@example.com", Username: "root@example.com", Role: entity.RoleSuper},
		{Subject: "local:admin@example.com", Email: "admin@example.com", Username: "admin@example.com", Role: entity.RoleAdmin},
		{Subject: "local:bob@example.com", Email: "bob@example.com", Username: "bob@example.com", Groups: []string{"devs"}, Role: entity.RoleMember},
		{Subject: "oidc:carol", Email: "carol@example.com", Role: entity.RoleMember, Disabled: true},
	}
	f.Perms.M["local:bob@example.com"] = entity.Permission{
		Connection: "demo", Member: "local:bob@example.com", MemberType: entity.MemberUser, Level: entity.LevelWrite,
	}
	return f
}

func TestDirectoryFiltersAndGuard(t *testing.T) {
	f := seed()
	super := &entity.Identity{Subject: "local:root@example.com", Role: entity.RoleSuper}
	admin := &entity.Identity{Subject: "local:admin@example.com", Role: entity.RoleAdmin}
	uc := user.List{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("GET", "/users", uc.Serve)

	for name, caller := range map[string]*entity.Identity{"super": super, "admin": admin} {
		rec := ucstest.Do(mux, http.MethodGet, "/users", "", caller)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "local:bob@example.com") {
			t.Fatalf("%s list: %d %s", name, rec.Code, rec.Body.String())
		}
	}

	rec := ucstest.Do(mux, http.MethodGet, "/users?q=bob@example.com", "", super)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "local:root@example.com") {
		t.Fatalf("search: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodGet, "/users?status=disabled", "", super)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "oidc:carol") || strings.Contains(rec.Body.String(), "local:bob@example.com") {
		t.Fatalf("disabled filter: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodGet, "/users", "", &entity.Identity{Subject: "local:bob@example.com", Role: entity.RoleMember})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a member must be refused: %d", rec.Code)
	}
}

func TestDetailCarriesAccessAndEvents(t *testing.T) {
	f := seed()
	admin := &entity.Identity{Subject: "local:admin@example.com", Role: entity.RoleAdmin}
	uc := user.Get{RegistryDeps: f.Deps()}
	rec := ucstest.Do(ucstest.Mux("GET", "/users/{subject}", uc.Serve), http.MethodGet, "/users/local:bob@example.com", "", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"access"`) || !strings.Contains(body, `"demo"`) || !strings.Contains(body, `"events"`) {
		t.Fatalf("detail body: %s", body)
	}
}

func TestRoleGuards(t *testing.T) {
	f := seed()
	super := &entity.Identity{Subject: "local:root@example.com", Role: entity.RoleSuper}
	uc := user.Update{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("PATCH", "/users/{subject}", uc.Serve)

	// the root is untouchable and nobody edits themselves
	for name, tc := range map[string]struct {
		subject string
		caller  *entity.Identity
	}{
		"super target": {"local:root@example.com", super},
		"self":         {"local:root@example.com", super},
	}{
		rec := ucstest.Do(mux, http.MethodPatch, "/users/"+tc.subject, `{"role":"member"}`, tc.caller)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s must be forbidden: %d", name, rec.Code)
		}
	}

	if rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{}`, super); rec.Code != http.StatusBadRequest {
		t.Fatalf("an empty patch must be a 400: %d", rec.Code)
	}
}

func TestRoleIsSuperOnly(t *testing.T) {
	f := seed()
	ctx := context.Background()
	super := &entity.Identity{Subject: "local:root@example.com", Role: entity.RoleSuper}
	admin := &entity.Identity{Subject: "local:admin@example.com", Role: entity.RoleAdmin}
	uc := user.Update{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("PATCH", "/users/{subject}", uc.Serve)

	rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"admin"}`, admin)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("an admin must not promote: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"admin"}`, super)
	if rec.Code != http.StatusOK {
		t.Fatalf("super promote: %d %s", rec.Code, rec.Body.String())
	}
	if u, _ := f.Users.GetUser(ctx, "local:bob@example.com"); u.Role != entity.RoleAdmin {
		t.Fatalf("bob role = %q, want admin", u.Role)
	}

	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"super"}`, super)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a second super must be a 400: %d", rec.Code)
	}
}
