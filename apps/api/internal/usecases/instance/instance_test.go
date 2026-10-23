package instance

import (
	"encoding/json"
	"net/http"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases/ucstest"
)

func TestGetLookup(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	c := f.Conns.M["demo"]
	c.ID, c.Password = "abc", "secret"
	f.Conns.M["demo"] = c

	mux := ucstest.Mux("GET", "/external/v1/instances/{id}", Get{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(mux, http.MethodGet, "/external/v1/instances/abc", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got entity.Connection
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Family != constants.FamilySQL || got.Password != "" {
		t.Fatalf("instance = %+v", got)
	}

	rec = ucstest.Do(mux, http.MethodGet, "/external/v1/instances/missing", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing id: status=%d want 404", rec.Code)
	}
}

func TestListSearch(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	f.Mods[constants.FamilyMemcache] = ucstest.Memcache{}
	f.Conns.M["prod-db"] = entity.Connection{Name: "prod-db", Driver: constants.DriverPostgres, Host: "db.internal", Port: 5432, User: "postgres", Database: "app"}
	f.Conns.M["cache"] = entity.Connection{Name: "cache", Driver: constants.DriverRedis, Host: "redis.internal", Port: 6379, Database: "0"}

	mux := ucstest.Mux("GET", "/external/v1/instances", List{RegistryDeps: f.Deps()}.Serve)
	var out struct {
		Instances []entity.Connection `json:"instances"`
		Total     int                 `json:"total"`
	}
	rec := ucstest.Do(mux, http.MethodGet, "/external/v1/instances?host=internal", "", nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Total != 2 {
		t.Fatalf("by host total=%d want 2", out.Total)
	}
	rec = ucstest.Do(mux, http.MethodGet, "/external/v1/instances?name=PROD", "", nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Total != 1 || out.Instances[0].Name != "prod-db" {
		t.Fatalf("by name = %+v", out)
	}
}

func TestRotatePassword(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	c := f.Conns.M["demo"]
	c.ID, c.Password = "abc", "old"
	f.Conns.M["demo"] = c

	mux := ucstest.Mux("POST", "/external/v1/instances/{id}/password", RotatePassword{RegistryDeps: f.Deps()}.Serve)
	admin := &entity.Identity{Subject: "root", Role: entity.RoleSuper}
	rec := ucstest.Do(mux, http.MethodPost, "/external/v1/instances/abc/password", `{"password":"new"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", rec.Code, rec.Body.String())
	}
	if f.Conns.M["demo"].Password != "new" {
		t.Fatal("password not rotated")
	}
	if rec := ucstest.Do(mux, http.MethodPost, "/external/v1/instances/abc/password", `{"password":""}`, admin); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty password status=%d want 400", rec.Code)
	}
}
