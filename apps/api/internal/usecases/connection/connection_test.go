package connection

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases/ucstest"
)

func TestGetRedactsPassword(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	c := f.Conns.M["demo"]
	c.Password = "supersecret"
	f.Conns.M["demo"] = c

	mux := ucstest.Mux("GET", "/external/v1/engine/{engine}/connections/{name}", Get{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections/demo", "", nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "supersecret") {
		t.Fatalf("redaction: %d %s", rec.Code, rec.Body.String())
	}
}

func TestListHidesUnreadable(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	f.Perms.M["readers"] = entity.Permission{Connection: "demo", Member: "readers", MemberType: entity.MemberGroup, Level: entity.LevelRead}

	mux := ucstest.Mux("GET", "/external/v1/engine/{engine}/connections", List{RegistryDeps: f.Deps()}.Serve)
	reader := &entity.Identity{Subject: "alice", Groups: []string{"readers"}}
	rec := ucstest.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections", "", reader)
	var out struct {
		Connections []entity.Connection `json:"connections"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Connections) != 1 {
		t.Fatalf("reader sees %d, want 1", len(out.Connections))
	}

	stranger := &entity.Identity{Subject: "bob", Groups: []string{"strangers"}}
	rec = ucstest.Do(mux, http.MethodGet, "/external/v1/engine/sql/connections", "", stranger)
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Connections) != 0 {
		t.Fatalf("stranger sees %d, want 0", len(out.Connections))
	}
}

func TestSaveRequiresAdmin(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	mux := ucstest.Mux("POST", "/external/v1/engine/{engine}/connections", Save{RegistryDeps: f.Deps()}.Serve)
	body := `{"name":"x","driver":"` + constants.DriverPostgres + `","host":"h","port":1,"user":"u","database":"d"}`
	rec := ucstest.Do(mux, http.MethodPost, "/external/v1/engine/sql/connections", body, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin save: %d %s", rec.Code, rec.Body.String())
	}
}
