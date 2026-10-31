package connection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"noxis/internal/common/testkit"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/mocks"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

// deps wires a use case to mocks; the test states the calls it expects.
func deps(t *testing.T) (usecases.RegistryDeps, *mocks.MockConnectionRepository, *mocks.MockPermissionRepository) {
	t.Helper()
	conns := mocks.NewMockConnectionRepository(gomock.NewController(t))
	perms := mocks.NewMockPermissionRepository(gomock.NewController(t))
	sq := mocks.NewMockSQLChangeEngine(gomock.NewController(t))
	sq.EXPECT().Driver().Return(constants.DriverPostgres).AnyTimes()
	sq.EXPECT().Supports(gomock.Any()).Return(true).AnyTimes()
	return usecases.RegistryDeps{
		Repositories: repositories.Registry{Connections: conns, Permissions: perms},
		Modules:      modules.New(map[string]interfaces.Module{constants.DriverPostgres: sq}),
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
	mux := testkit.Mux("GET", "/external/v1/engine/{driver}/connections/{name}", Get{RegistryDeps: d}.Serve)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/engine/postgres/connections/demo", "", nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "supersecret") {
		t.Fatalf("redaction: %d %s", rec.Code, rec.Body.String())
	}
}

func TestListHidesUnreadable(t *testing.T) {
	d, conns, perms := deps(t)
	c := demo()
	conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{c}, nil).Times(2)
	mux := testkit.Mux("GET", "/external/v1/engine/{driver}/connections", List{RegistryDeps: d}.Serve)

	reader := &entity.Identity{Subject: "alice", Groups: []string{"readers"}}
	perms.EXPECT().ListPermissions(gomock.Any(), "demo").Return([]entity.Permission{{
		Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead,
	}}, nil)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/engine/postgres/connections", "", reader)
	var out entity.ConnectionList
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Connections) != 1 {
		t.Fatalf("reader sees %d, want 1", len(out.Connections))
	}

	stranger := &entity.Identity{Subject: "bob", Groups: []string{"strangers"}}
	perms.EXPECT().ListPermissions(gomock.Any(), "demo").Return([]entity.Permission{{
		Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead,
	}}, nil)
	rec = testkit.Do(mux, http.MethodGet, "/external/v1/engine/postgres/connections", "", stranger)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Connections) != 0 {
		t.Fatalf("stranger sees %d, want 0", len(out.Connections))
	}
}

func TestSaveRequiresAdmin(t *testing.T) {
	d, _, _ := deps(t)
	mux := testkit.Mux("POST", "/external/v1/engine/{driver}/connections", Save{RegistryDeps: d}.Serve)
	body := `{"name":"x","driver":"` + constants.DriverPostgres + `","host":"h","port":1,"user":"u","database":"d"}`
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/engine/postgres/connections", body, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin save: %d %s", rec.Code, rec.Body.String())
	}
}

// TestProbeSavesNothing proves the draft is pinged without the repository ever being read or written: the mock has no
// expectation for it, so any call would fail the test.
// super is the only caller allowed to probe: the route opens a socket to a host the caller names.
func super() *entity.Identity { return &entity.Identity{Subject: "root", Role: entity.RoleSuper} }

func TestProbeSavesNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	conns := mocks.NewMockConnectionRepository(ctrl)
	sq := mocks.NewMockSQLChangeEngine(ctrl)
	sq.EXPECT().Driver().Return(constants.DriverPostgres).AnyTimes()
	sq.EXPECT().Supports(gomock.Any()).Return(true).AnyTimes()
	sq.EXPECT().Ping(gomock.Any(), gomock.Any()).Return(nil)
	d := usecases.RegistryDeps{
		Repositories: repositories.Registry{Connections: conns},
		Modules:      modules.New(map[string]interfaces.Module{constants.DriverPostgres: sq}),
	}
	body, err := json.Marshal(demo())
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	mux := testkit.Mux("POST", "/external/v1/engine/{driver}/connections/probe", Probe{RegistryDeps: d}.Serve)
	rec := testkit.DoContext(context.Background(), mux, http.MethodPost, "/external/v1/engine/postgres/connections/probe", string(body), super())
	if rec.Code != http.StatusOK {
		t.Fatalf("probe status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestProbeReportsTheProbeError keeps the module's message, because the form shows it to the caller.
func TestProbeReportsTheProbeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	conns := mocks.NewMockConnectionRepository(ctrl)
	sq := mocks.NewMockSQLChangeEngine(ctrl)
	sq.EXPECT().Driver().Return(constants.DriverPostgres).AnyTimes()
	sq.EXPECT().Supports(gomock.Any()).Return(true).AnyTimes()
	sq.EXPECT().Ping(gomock.Any(), gomock.Any()).Return(errors.New("connection refused"))
	d := usecases.RegistryDeps{
		Repositories: repositories.Registry{Connections: conns},
		Modules:      modules.New(map[string]interfaces.Module{constants.DriverPostgres: sq}),
	}
	body, err := json.Marshal(demo())
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	mux := testkit.Mux("POST", "/external/v1/engine/{driver}/connections/probe", Probe{RegistryDeps: d}.Serve)
	rec := testkit.DoContext(context.Background(), mux, http.MethodPost, "/external/v1/engine/postgres/connections/probe", string(body), super())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("failed probe: %d %s", rec.Code, rec.Body.String())
	}
}
