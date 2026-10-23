package access_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases/access"
	"noxis/internal/usecases/ucstest"
)

func fixture() *ucstest.Fixture {
	f := ucstest.New(&ucstest.SQL{})
	f.Conns.M["demo"] = entity.Connection{Name: "demo", Driver: constants.DriverPostgres, Family: constants.FamilySQL}
	f.Users.Accounts = []entity.User{
		{Subject: "local:root", Username: "root", Role: entity.RoleSuper},
		{Subject: "local:bob", Username: "bob"},
		{Subject: "local:alice", Username: "alice"},
	}
	// alice is the instance admin
	f.Perms.M["local:alice"] = entity.Permission{
		Connection: "demo", Member: "local:alice", MemberType: entity.MemberUser,
		Level: entity.LevelAdmin, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	return f
}

func TestRequestAndApproveFlow(t *testing.T) {
	f := fixture()
	bob := &entity.Identity{Subject: "local:bob"}
	alice := &entity.Identity{Subject: "local:alice"}

	rec := ucstest.Do(ucstest.Mux("POST", "/access-requests", access.Create{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/access-requests", `{"connection":"demo","level":"write","reason":"need it"}`, bob)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"pending"`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	id := f.Requests.Rows[0].ID

	approve := ucstest.Mux("POST", "/access-requests/{id}/approve", access.Approve{RegistryDeps: f.Deps()}.Serve)

	// the requester cannot approve their own request
	rec = ucstest.Do(approve, http.MethodPost, "/access-requests/"+id+"/approve", `{}`, bob)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("self-approval: %d %s", rec.Code, rec.Body.String())
	}

	// the instance admin can
	rec = ucstest.Do(approve, http.MethodPost, "/access-requests/"+id+"/approve", `{"note":"ok"}`, alice)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", rec.Code, rec.Body.String())
	}
	if got := f.Perms.M["local:bob"]; got.Level != entity.LevelWrite || got.ExpiresAt == 0 {
		t.Fatalf("grant not written: %+v", got)
	}
	if f.Requests.Rows[0].Status != entity.RequestApproved || f.Requests.Rows[0].DecidedBy != "local:alice" {
		t.Fatalf("decision not stored: %+v", f.Requests.Rows[0])
	}
}

func TestToApproveScope(t *testing.T) {
	f := fixture()
	bob := &entity.Identity{Subject: "local:bob"}
	alice := &entity.Identity{Subject: "local:alice"}

	if rec := ucstest.Do(ucstest.Mux("POST", "/access-requests", access.Create{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/access-requests", `{"connection":"demo","level":"read"}`, bob); rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}

	list := ucstest.Mux("GET", "/access-requests", access.List{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(list, http.MethodGet, "/access-requests?scope=to-approve", "", bob)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "local:bob") {
		t.Fatalf("requester must not see the inbox: %d %s", rec.Code, rec.Body.String())
	}
	rec = ucstest.Do(list, http.MethodGet, "/access-requests?scope=to-approve", "", alice)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "local:bob") {
		t.Fatalf("instance admin must see it: %d %s", rec.Code, rec.Body.String())
	}
}
