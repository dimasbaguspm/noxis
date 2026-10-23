package connection

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

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

// deps wires a use case to mocks; the test states the calls it expects.
func deps(t *testing.T) (usecases.RegistryDeps, *mocks.MockConnectionRepository, *mocks.MockPermissionRepository) {
	t.Helper()
	conns := mocks.NewMockConnectionRepository(gomock.NewController(t))
	perms := mocks.NewMockPermissionRepository(gomock.NewController(t))
	sq := mocks.NewMockSQLModule(gomock.NewController(t))
	sq.EXPECT().Family().Return(constants.FamilySQL).AnyTimes()
	sq.EXPECT().Supports(gomock.Any()).Return(true).AnyTimes()
	return usecases.RegistryDeps{
		Registry: repositories.Registry{Connections: conns, Permissions: perms},
		Modules:  modules.New(map[string]interfaces.Module{constants.FamilySQL: sq}, nil),
	}, conns, perms
}

// demo is the connection these tests act on.
func demo() entity.Connection {
	return entity.Connection{
		Name: "demo", Driver: constants.DriverPostgres, Host: "db", Port: 5432,
		User: "postgres", Database: "demo", Password: "supersecret",
	}
}

func TestGetRedactsPassword(t *testing.T) {
	d, conns, _ := deps(t)
	conns.EXPECT().Get(gomock.Any(), "demo").Return(&entity.Connection{
		Name: "demo", Driver: constants.DriverPostgres, Password: "supersecret",
	}, nil)
	// a super admin reads without a grant lookup
	mux := testkit.Mux("GET", "/external/v1/engine/{engine}/connections/{name}", Get{RegistryDeps: d}.Serve)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections/demo", "", nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "supersecret") {
		t.Fatalf("redaction: %d %s", rec.Code, rec.Body.String())
	}
}

func TestListHidesUnreadable(t *testing.T) {
	d, conns, perms := deps(t)
	c := demo()
	conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{c}, nil).Times(2)
	mux := testkit.Mux("GET", "/external/v1/engine/{engine}/connections", List{RegistryDeps: d}.Serve)

	reader := &entity.Identity{Subject: "alice", Groups: []string{"readers"}}
	perms.EXPECT().ListPermissions(gomock.Any(), "demo").Return([]entity.Permission{{
		Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead,
	}}, nil)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections", "", reader)
	var out entity.ConnectionList
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Connections) != 1 {
		t.Fatalf("reader sees %d, want 1", len(out.Connections))
	}

	stranger := &entity.Identity{Subject: "bob", Groups: []string{"strangers"}}
	perms.EXPECT().ListPermissions(gomock.Any(), "demo").Return([]entity.Permission{{
		Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead,
	}}, nil)
	rec = testkit.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections", "", stranger)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Connections) != 0 {
		t.Fatalf("stranger sees %d, want 0", len(out.Connections))
	}
}

func TestSaveRequiresAdmin(t *testing.T) {
	d, _, _ := deps(t)
	mux := testkit.Mux("POST", "/external/v1/engine/{engine}/connections", Save{RegistryDeps: d}.Serve)
	body := `{"name":"x","driver":"` + constants.DriverPostgres + `","host":"h","port":1,"user":"u","database":"d"}`
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/engine/sql/connections", body, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin save: %d %s", rec.Code, rec.Body.String())
	}
}
