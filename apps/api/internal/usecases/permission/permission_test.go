package permission

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases/ucstest"
)

func TestListRequiresInstanceAdmin(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	mux := ucstest.Mux("GET", "/external/v1/engine/{engine}/connections/{name}/permissions", List{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no principal: %d %s", rec.Code, rec.Body.String())
	}
	// a member with no grant on the connection is forbidden
	rec = ucstest.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", &entity.Identity{Subject: "local:bob"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin list: %d %s", rec.Code, rec.Body.String())
	}
}

func TestCRUD(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	admin := &entity.Identity{Subject: "root", Super: true}

	put := ucstest.Mux("PUT", "/external/v1/engine/{engine}/connections/{name}/permissions", Set{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(put, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"write"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status=%d body=%s", rec.Code, rec.Body.String())
	}

	get := ucstest.Mux("GET", "/external/v1/engine/{engine}/connections/{name}/permissions", List{RegistryDeps: f.Deps()}.Serve)
	rec = ucstest.Do(get, http.MethodGet, "/external/v1/engine/sql/connections/demo/permissions", "", admin)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"member":"dev"`) {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(put, http.MethodPut, "/external/v1/engine/sql/connections/demo/permissions", `{"member":"dev","memberType":"group","level":"root"}`, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad level status=%d want 400", rec.Code)
	}

	del := ucstest.Mux("DELETE", "/external/v1/engine/{engine}/connections/{name}/permissions", Delete{RegistryDeps: f.Deps()}.Serve)
	rec = ucstest.Do(del, http.MethodDelete, "/external/v1/engine/sql/connections/demo/permissions?member=dev", "", admin)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMembers(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	f.Perms.M["readers"] = entity.Permission{Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead}
	f.Users.Accounts = []entity.User{{Subject: "local:alice", Username: "alice"}}

	rec := ucstest.Do(ucstest.Mux("GET", "/external/v1/config/members", Members{RegistryDeps: f.Deps()}.Serve),
		http.MethodGet, "/external/v1/config/members", "", &entity.Identity{Subject: "root", Super: true})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"readers"`) || !strings.Contains(rec.Body.String(), `"local:alice"`) {
		t.Fatalf("members status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInstanceAdminCanReadAndRevokeButNotGrant(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	f.Conns.M["demo"] = entity.Connection{Name: "demo", Driver: constants.DriverPostgres, Family: constants.FamilySQL}
	f.Perms.M["local:bob"] = entity.Permission{
		Connection: "demo", Member: "local:bob", MemberType: entity.MemberUser,
		Level: entity.LevelAdmin, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	f.Perms.M["dev"] = entity.Permission{
		Connection: "demo", Member: "dev", MemberType: entity.MemberGroup,
		Level: entity.LevelRead, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	bob := &entity.Identity{Subject: "local:bob"}

	rec := ucstest.Do(ucstest.Mux("GET", "/x/{engine}/c/{name}", List{RegistryDeps: f.Deps()}.Serve),
		http.MethodGet, "/x/sql/c/demo", "", bob)
	if rec.Code != http.StatusOK {
		t.Fatalf("instance admin list: %d %s", rec.Code, rec.Body.String())
	}

	rec = ucstest.Do(ucstest.Mux("PUT", "/x/{engine}/c/{name}", Set{RegistryDeps: f.Deps()}.Serve),
		http.MethodPut, "/x/sql/c/demo", `{"member":"x","memberType":"group","level":"read"}`, bob)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("instance admin must not grant: %d %s", rec.Code, rec.Body.String())
	}

	// the instance's own admin may take access away
	rec = ucstest.Do(ucstest.Mux("DELETE", "/x/{engine}/c/{name}", Delete{RegistryDeps: f.Deps()}.Serve),
		http.MethodDelete, "/x/sql/c/demo?member=dev", "", bob)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("instance admin revoke: %d %s", rec.Code, rec.Body.String())
	}

	// a member with no admin grant on the connection may not
	carol := &entity.Identity{Subject: "local:carol"}
	rec = ucstest.Do(ucstest.Mux("DELETE", "/x/{engine}/c/{name}", Delete{RegistryDeps: f.Deps()}.Serve),
		http.MethodDelete, "/x/sql/c/demo?member=dev", "", carol)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin must not revoke: %d %s", rec.Code, rec.Body.String())
	}
}
