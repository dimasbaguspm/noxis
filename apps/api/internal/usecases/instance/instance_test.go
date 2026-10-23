package instance

import (
	"context"
	"encoding/json"
	"net/http"
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

// deps wires the instance use cases to mocks with both engine families present.
func deps(t *testing.T) (usecases.RegistryDeps, *mocks.MockConnectionRepository) {
	t.Helper()
	conns := mocks.NewMockConnectionRepository(gomock.NewController(t))
	sq := mocks.NewMockSQLModule(gomock.NewController(t))
	sq.EXPECT().Family().Return(constants.FamilySQL).AnyTimes()
	sq.EXPECT().Supports(constants.DriverPostgres).Return(true).AnyTimes()
	sq.EXPECT().Supports(gomock.Any()).Return(false).AnyTimes()
	mc := mocks.NewMockMemcacheModule(gomock.NewController(t))
	mc.EXPECT().Family().Return(constants.FamilyMemcache).AnyTimes()
	mc.EXPECT().Supports(constants.DriverRedis).Return(true).AnyTimes()
	mc.EXPECT().Supports(gomock.Any()).Return(false).AnyTimes()
	return usecases.RegistryDeps{
		Registry: repositories.Registry{Connections: conns},
		Modules: modules.New(map[string]interfaces.Module{
			constants.FamilySQL: sq, constants.FamilyMemcache: mc,
		}, interfaces.ChangeFlows{}, nil),
	}, conns
}

func TestGetLookup(t *testing.T) {
	d, conns := deps(t)
	conns.EXPECT().GetByID(gomock.Any(), "abc").Return(&entity.Connection{
		ID: "abc", Name: "demo", Driver: constants.DriverPostgres, Password: "secret",
	}, nil)
	conns.EXPECT().GetByID(gomock.Any(), "missing").Return(nil, constants.ErrNotFound)

	mux := testkit.Mux("GET", "/external/v1/instances/{id}", Get{RegistryDeps: d}.Serve)
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/instances/abc", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got entity.Connection
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Family != constants.FamilySQL || got.Password != "" {
		t.Fatalf("instance = %+v", got)
	}

	rec = testkit.Do(mux, http.MethodGet, "/external/v1/instances/missing", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing id: status=%d want 404", rec.Code)
	}
}

func TestListSearch(t *testing.T) {
	d, conns := deps(t)
	conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{
		{Name: "prod-db", Driver: constants.DriverPostgres, Host: "db.internal", Port: 5432, User: "postgres", Database: "app"},
		{Name: "cache", Driver: constants.DriverRedis, Host: "redis.internal", Port: 6379, Database: "0"},
	}, nil).AnyTimes()

	mux := testkit.Mux("GET", "/external/v1/instances", List{RegistryDeps: d}.Serve)
	var out entity.InstanceList
	rec := testkit.Do(mux, http.MethodGet, "/external/v1/instances?host=internal", "", nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Total != 2 {
		t.Fatalf("by host total=%d want 2", out.Total)
	}
	rec = testkit.Do(mux, http.MethodGet, "/external/v1/instances?name=PROD", "", nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Total != 1 || out.Instances[0].Name != "prod-db" {
		t.Fatalf("by name = %+v", out)
	}
}

func TestRotatePassword(t *testing.T) {
	d, conns := deps(t)
	conns.EXPECT().GetByID(gomock.Any(), "abc").Return(&entity.Connection{
		ID: "abc", Name: "demo", Driver: constants.DriverPostgres, Password: "old",
	}, nil)
	conns.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c entity.Connection) error {
		if c.Password != "new" {
			t.Fatalf("saved password = %q, want new", c.Password)
		}
		return nil
	})

	mux := testkit.Mux("POST", "/external/v1/instances/{id}/password", RotatePassword{RegistryDeps: d}.Serve)
	admin := &entity.Identity{Subject: "root", Role: entity.RoleSuper}
	rec := testkit.Do(mux, http.MethodPost, "/external/v1/instances/abc/password", `{"password":"new"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := testkit.Do(mux, http.MethodPost, "/external/v1/instances/abc/password", `{"password":""}`, admin); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty password status=%d want 400", rec.Code)
	}
}
