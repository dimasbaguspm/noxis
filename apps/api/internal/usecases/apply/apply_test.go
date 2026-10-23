package apply

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases/ucstest"
)

func approved(id string) entity.Change {
	return entity.Change{ID: id, Connection: "demo", Kind: constants.KindDDL, Status: constants.ChangeApproved, Query: "CREATE TABLE things (id INT)"}
}

func TestApplyAfterApprove(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	f.Changes.M["x"] = approved("x")

	mux := ucstest.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/apply", Apply{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(mux, http.MethodPost, "/external/v1/engine/sql/changes/x/apply", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := f.Changes.M["x"].Status; got != constants.ChangeApplied {
		t.Fatalf("status=%q want applied", got)
	}
}

func TestApplyFailureIsRecorded(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{ApplyErr: errors.New("statement 1: duplicate key")})
	f.Changes.M["x"] = approved("x")

	mux := ucstest.Mux("POST", "/external/v1/engine/{engine}/changes/{id}/apply", Apply{RegistryDeps: f.Deps()}.Serve)
	rec := ucstest.Do(mux, http.MethodPost, "/external/v1/engine/sql/changes/x/apply", "", &entity.Identity{Subject: "root", Role: entity.RoleSuper})
	if rec.Code == http.StatusOK {
		t.Fatalf("apply must surface the failure: %s", rec.Body.String())
	}
	stored := f.Changes.M["x"]
	if stored.Status != constants.ChangeFailed {
		t.Fatalf("failure must be recorded: %+v", stored)
	}
	last := stored.Checks[len(stored.Checks)-1]
	if last.OK || !strings.Contains(last.Error, "duplicate key") {
		t.Fatalf("apply check: %+v", last)
	}
}

func TestTickAppliesDueScheduledChange(t *testing.T) {
	f := ucstest.New(&ucstest.SQL{})
	due := approved("due")
	due.ScheduledAt = time.Now().Add(-time.Minute).Unix()
	f.Changes.M["due"] = due
	f.Changes.M["wait"] = approved("wait") // immediate: not due

	rec := ucstest.Do(ucstest.Mux("POST", "/internal/callback/scheduler", Tick{RegistryDeps: f.Deps()}.Serve),
		http.MethodPost, "/internal/callback/scheduler", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"due"`) {
		t.Fatalf("tick status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := f.Changes.M["due"].Status; got != constants.ChangeApplied {
		t.Fatalf("due status=%q want applied", got)
	}
	if got := f.Changes.M["wait"].Status; got != constants.ChangeApproved {
		t.Fatalf("immediate status=%q want approved", got)
	}
}
