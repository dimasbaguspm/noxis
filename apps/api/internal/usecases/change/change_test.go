package change

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/modules/sql"
	"noxis/internal/usecases/ucstest"
)

func admin() *entity.Identity { return &entity.Identity{Subject: "root", Role: entity.RoleSuper} }

const sampleSQL = `{"connection":"demo","notes":"add things","query":"CREATE TABLE things (id INT)","rollback":"DROP TABLE things"}`

func TestFilterByConnection(t *testing.T) {
	changes := []entity.Change{{Connection: "prod-db"}, {Connection: "cache"}, {Connection: "prod-cache"}}
	cases := []struct {
		needle string
		want   int
	}{
		{"", 3},
		{"PROD", 2},
		{"cache", 2},
		{"nope", 0},
	}
	for _, tc := range cases {
		if got := filterByConnection(changes, tc.needle); len(got) != tc.want {
			t.Fatalf("filter %q = %d, want %d", tc.needle, len(got), tc.want)
		}
	}
}

func TestSubmitThenApprove(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})

	rec := ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes", Submit{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes", sampleSQL, admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", rec.Code, rec.Body.String())
	}
	var ch entity.Change
	if err := json.Unmarshal(rec.Body.Bytes(), &ch); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ch.Status != constants.ChangeValidated || ch.ID == "" || len(ch.Checks) != 2 {
		t.Fatalf("submit result: %+v", ch)
	}

	rec = ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/approve", Approve{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes/"+ch.ID+"/approve", "", admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", rec.Code, rec.Body.String())
	}
	var approved entity.Change
	_ = json.Unmarshal(rec.Body.Bytes(), &approved)
	if approved.Status != constants.ChangeApproved || approved.ApprovedAt == 0 {
		t.Fatalf("approve result: %+v", approved)
	}
}

func TestSubmitValidationFailures(t *testing.T) {
	// a read on the change path is refused up front
	f := ucstest.New(&ucstest.SQL{WriteErr: errors.New(`statement "SELECT" not allowed here`)})
	rec := ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes", Submit{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes", sampleSQL, admin())
	var failed entity.Change
	_ = json.Unmarshal(rec.Body.Bytes(), &failed)
	if failed.Status != constants.ChangeFailed || failed.ID != "" {
		t.Fatalf("write failure: %+v", failed)
	}

	// a dry-run failure lands as failed with the reason recorded
	f2 := ucstest.New(&ucstest.SQL{ValidateErr: errors.New("rollback statement 1: syntax error")})
	rec = ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes", Submit{RegistryDeps: f2.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes", sampleSQL, admin())
	_ = json.Unmarshal(rec.Body.Bytes(), &failed)
	if failed.Status != constants.ChangeFailed || len(failed.Checks) < 2 || !strings.Contains(failed.Checks[1].Error, "syntax error") {
		t.Fatalf("dry-run failure: %+v", failed)
	}

	// mysql DDL cannot be dry-run: unverified, still approvable
	f3 := ucstest.New(&ucstest.SQL{ValidateErr: sql.ErrUnverifiable})
	rec = ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes", Submit{RegistryDeps: f3.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes", sampleSQL, admin())
	var unverified entity.Change
	_ = json.Unmarshal(rec.Body.Bytes(), &unverified)
	if unverified.Status != constants.ChangeUnverified {
		t.Fatalf("unverified: %+v", unverified)
	}
	rec = ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/approve", Approve{RegistryDeps: f3.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes/"+unverified.ID+"/approve", "", admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("unverified must be approvable: %d %s", rec.Code, rec.Body.String())
	}
}

func TestKindGated(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	f.Changes.M["m1"] = entity.Change{ID: "m1", Kind: constants.KindMongo, Status: constants.ChangeValidated, Connection: "demo"}
	rec := ucstest.Do(ucstest.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/approve", Approve{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/external/v1/engine/sql/changes/m1/approve", "", admin())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sql engine must not approve a mongo change: %d %s", rec.Code, rec.Body.String())
	}
}
