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
		{Subject: "local:root", Username: "root", Super: true},
		{Subject: "local:bob", Username: "bob", Groups: []string{"devs"}},
		{Subject: "oidc:carol", Disabled: true},
	}
	f.Perms.M["local:bob"] = entity.Permission{
		Connection: "demo", Member: "local:bob", MemberType: entity.MemberUser, Level: entity.LevelWrite,
	}
	return f
}

func TestDirectoryFiltersAndGuard(t *testing.T) {
	f := seed()
	super := &entity.Identity{Subject: "local:root", Super: true}
	uc := user.List{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("GET", "/users", uc.Serve)

	rec := ucstest.Do(mux, http.MethodGet, "/users", "", super)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "local:bob") {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodGet, "/users?q=bob", "", super)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "local:root") {
		t.Fatalf("search: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodGet, "/users?status=disabled", "", super)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "oidc:carol") || strings.Contains(rec.Body.String(), "local:bob") {
		t.Fatalf("status filter: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(mux, http.MethodGet, "/users", "", &entity.Identity{Subject: "local:bob"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-super must be refused: %d", rec.Code)
	}
}

func TestDetailCarriesAccessAndEvents(t *testing.T) {
	f := seed()
	super := &entity.Identity{Subject: "local:root", Super: true}
	uc := user.Get{RegistryDeps: f.Deps()}
	rec := ucstest.Do(ucstest.Mux("GET", "/users/{subject}", uc.Serve), http.MethodGet, "/users/local:bob", "", super)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"access"`) || !strings.Contains(body, `"demo"`) || !strings.Contains(body, `"events"`) {
		t.Fatalf("detail body: %s", body)
	}
}

func TestDisableGuards(t *testing.T) {
	f := seed()
	ctx := context.Background()
	super := &entity.Identity{Subject: "local:root", Super: true}
	uc := user.Update{RegistryDeps: f.Deps()}
	mux := ucstest.Mux("PATCH", "/users/{subject}", uc.Serve)

	rec := ucstest.Do(mux, http.MethodPatch, "/users/local:bob", `{"disabled":true}`, super)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable: %d %s", rec.Code, rec.Body.String())
	}
	if u, _ := f.Users.GetUser(ctx, "local:bob"); !u.Disabled {
		t.Fatal("bob must be disabled")
	}

	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:root", `{"disabled":true}`, super)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("the super admin must not be disableable: %d", rec.Code)
	}

	rec = ucstest.Do(mux, http.MethodPatch, "/users/local:bob", `{}`, super)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing disabled must be a 400: %d", rec.Code)
	}
}
