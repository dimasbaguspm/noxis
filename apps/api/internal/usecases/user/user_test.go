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
		{Subject: "local:root@example.com", Email: "root@example.com", Username: "root@example.com", Role: entity.RoleSuper, Status: entity.StatusActive},
		{Subject: "local:admin@example.com", Email: "admin@example.com", Username: "admin@example.com", Role: entity.RoleAdmin, Status: entity.StatusActive},
		{Subject: "local:bob@example.com", Email: "bob@example.com", Username: "bob@example.com", Groups: []string{"devs"}, Role: entity.RoleMember, Status: entity.StatusActive},
		{Subject: "oidc:carol", Email: "carol@example.com", Role: entity.RoleMember, Status: entity.StatusBanned},
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

	rec = ucstest.Do(mux, http.MethodGet, "/users?status=banned", "", super)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "oidc:carol") || strings.Contains(rec.Body.String(), "local:bob@example.com") {
		t.Fatalf("status filter: %d %s", rec.Code, rec.Body.String())
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

func TestStatusGuards(t *testing.T) {
	f := seed()
	ctx := context.Background()
	super := &entity.Identity{Subject: "local:root@example.com", Role: entity.RoleSuper}
	admin := &entity.Identity{Subject: "local:admin@example.com", Role: entity.RoleAdmin}
	uc := user.Update{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("PATCH", "/users/{subject}", uc.Serve)

	rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"status":"banned"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ban: %d %s", rec.Code, rec.Body.String())
	}
	if u, _ := f.Users.GetUser(ctx, "local:bob@example.com"); u.Active() {
		t.Fatal("bob must be banned")
	}

	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"status":"active"}`, admin)
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
		rec := ucstest.Do(mux, http.MethodPatch, "/users/"+tc.subject, `{"status":"banned"}`, tc.caller)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s must be forbidden: %d", name, rec.Code)
		}
	}

	// an admin may not touch a peer admin
	f.Users.Accounts = append(f.Users.Accounts, entity.User{
		Subject: "local:other@example.com", Email: "other@example.com", Role: entity.RoleAdmin, Status: entity.StatusActive,
	})
	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:other@example.com", `{"status":"banned"}`, admin)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin banning a peer must be forbidden: %d", rec.Code)
	}
	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:other@example.com", `{"status":"banned"}`, super)
	if rec.Code != http.StatusOK {
		t.Fatalf("super banning an admin: %d %s", rec.Code, rec.Body.String())
	}

	if rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{}`, super); rec.Code != http.StatusBadRequest {
		t.Fatalf("an empty patch must be a 400: %d", rec.Code)
	}
	if rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"status":"root"}`, super); rec.Code != http.StatusBadRequest {
		t.Fatalf("an unknown status must be a 400: %d", rec.Code)
	}
}

// a patch whose second field is invalid must not apply its first field
func TestPatchValidatesBeforeMutating(t *testing.T) {
	f := seed()
	super := &entity.Identity{Subject: "local:root@example.com", Role: entity.RoleSuper}
	uc := user.Update{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("PATCH", "/users/{subject}", uc.Serve)

	rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob@example.com", `{"role":"admin","status":"root"}`, super)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid patch must be a 400: %d", rec.Code)
	}
	if u, _ := f.Users.GetUser(context.Background(), "local:bob@example.com"); u.Role == entity.RoleAdmin {
		t.Fatal("the role must not be applied when the status is invalid")
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


