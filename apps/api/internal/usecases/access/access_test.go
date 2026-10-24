package access_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"noxis/internal/common/testkit"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/mocks"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
	"noxis/internal/usecases/access"
)

// harness is the access-request world: one connection, alice holding admin on it, and an in-memory policy built on the repository mocks.
type harness struct {
	deps     usecases.RegistryDeps
	grants   map[string]entity.Permission
	requests []entity.AccessRequest
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &harness{grants: map[string]entity.Permission{}}
	conns := mocks.NewMockConnectionRepository(ctrl)
	perms := mocks.NewMockPermissionRepository(ctrl)
	requests := mocks.NewMockAccessRequestRepository(ctrl)

	conns.EXPECT().Get(gomock.Any(), "demo").Return(&entity.Connection{
		Name: "demo", Driver: constants.DriverPostgres, Family: constants.FamilySQL,
	}, nil).AnyTimes()
	conns.EXPECT().List(gomock.Any()).Return([]entity.Connection{{Name: "demo"}}, nil).AnyTimes()

	perms.EXPECT().ListPermissions(gomock.Any(), "demo").DoAndReturn(func(context.Context, string) ([]entity.Permission, error) {
		out := make([]entity.Permission, 0, len(h.grants))
		for _, p := range h.grants {
			out = append(out, p)
		}
		return out, nil
	}).AnyTimes()
	perms.EXPECT().SetPermission(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p entity.Permission) error {
		h.grants[p.Member] = p
		return nil
	}).AnyTimes()

	requests.EXPECT().PendingRequestExists(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, subject, connection string) (bool, error) {
			for i := range h.requests {
				r := &h.requests[i]
				if r.Subject == subject && r.Connection == connection && r.Status == entity.RequestPending {
					return true, nil
				}
			}
			return false, nil
		}).AnyTimes()
	requests.EXPECT().CreateAccessRequest(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r entity.AccessRequest) error {
		h.requests = append(h.requests, r)
		return nil
	}).AnyTimes()
	requests.EXPECT().GetAccessRequest(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id string) (*entity.AccessRequest, error) {
		for i := range h.requests {
			if h.requests[i].ID == id {
				r := h.requests[i]
				return &r, nil
			}
		}
		return nil, constants.ErrNotFound
	}).AnyTimes()
	requests.EXPECT().ListPendingRequests(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, connections []string) ([]entity.AccessRequest, error) {
			allowed := map[string]struct{}{}
			for _, c := range connections {
				allowed[c] = struct{}{}
			}
			out := []entity.AccessRequest{}
			for i := range h.requests {
				r := &h.requests[i]
				if r.Status != entity.RequestPending {
					continue
				}
				if connections != nil {
					if _, ok := allowed[r.Connection]; !ok {
						continue
					}
				}
				out = append(out, *r)
			}
			return out, nil
		}).AnyTimes()
	requests.EXPECT().SetAccessRequestDecision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id, status, decidedBy string, decidedAt int64, note string) error {
			for i := range h.requests {
				if h.requests[i].ID == id {
					h.requests[i].Status, h.requests[i].DecidedBy = status, decidedBy
					h.requests[i].DecidedAt, h.requests[i].DecisionNote = decidedAt, note
					return nil
				}
			}
			return constants.ErrNotFound
		}).AnyTimes()

	h.deps = usecases.RegistryDeps{Repositories: repositories.Registry{
		Connections: conns, Permissions: perms, AccessRequests: requests,
	}}
	// alice is the instance admin
	h.grants["local:alice"] = entity.Permission{
		Connection: "demo", Member: "local:alice", MemberType: entity.MemberUser,
		Level: entity.LevelAdmin, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	return h
}

func TestRequestAndApproveFlow(t *testing.T) {
	h := newHarness(t)
	bob := &entity.Identity{Subject: "local:bob"}
	alice := &entity.Identity{Subject: "local:alice"}

	rec := testkit.Do(testkit.Mux("POST", "/access-requests", access.Create{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/access-requests", `{"connection":"demo","level":"write","reason":"need it"}`, bob)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"pending"`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	id := h.requests[0].ID

	approve := testkit.Mux("POST", "/access-requests/{id}/approve", access.Approve{RegistryDeps: h.deps}.Serve)

	// the requester cannot approve their own request
	rec = testkit.Do(approve, http.MethodPost, "/access-requests/"+id+"/approve", `{}`, bob)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("self-approval: %d %s", rec.Code, rec.Body.String())
	}

	// the instance admin can
	rec = testkit.Do(approve, http.MethodPost, "/access-requests/"+id+"/approve", `{"note":"ok"}`, alice)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", rec.Code, rec.Body.String())
	}
	if got := h.grants["local:bob"]; got.Level != entity.LevelWrite || got.ExpiresAt == 0 {
		t.Fatalf("grant not written: %+v", got)
	}
	if h.requests[0].Status != entity.RequestApproved || h.requests[0].DecidedBy != "local:alice" {
		t.Fatalf("decision not stored: %+v", h.requests[0])
	}
}

func TestToApproveScope(t *testing.T) {
	h := newHarness(t)
	bob := &entity.Identity{Subject: "local:bob"}
	alice := &entity.Identity{Subject: "local:alice"}

	if rec := testkit.Do(testkit.Mux("POST", "/access-requests", access.Create{RegistryDeps: h.deps}.Serve),
		http.MethodPost, "/access-requests", `{"connection":"demo","level":"read"}`, bob); rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}

	list := testkit.Mux("GET", "/access-requests", access.List{RegistryDeps: h.deps}.Serve)
	rec := testkit.Do(list, http.MethodGet, "/access-requests?scope=to-approve", "", bob)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "local:bob") {
		t.Fatalf("requester must not see the inbox: %d %s", rec.Code, rec.Body.String())
	}
	rec = testkit.Do(list, http.MethodGet, "/access-requests?scope=to-approve", "", alice)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "local:bob") {
		t.Fatalf("instance admin must see it: %d %s", rec.Code, rec.Body.String())
	}
}
