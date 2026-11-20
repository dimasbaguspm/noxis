package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"noxis/internal/app/proposal"
	"noxis/internal/app/read"
	"noxis/internal/app/target"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"noxis/internal/httpapi/middlewares"
	"strings"
	"testing"
	"time"
)

const (
	targetRef   = "0192f0c0-0000-7000-8000-000000000001"
	proposalRef = "0192f0c0-0000-7000-8000-000000000002"
)

// newHandlers builds the transport over the memory store: the handlers only ever see use cases, so
// the wiring is the same shape as cmd/http/wiring.go.
func newHandlers(s *store) *Handlers {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	driver := testDriver{s: s}
	targetDeps := target.Deps{Targets: s, Journal: s, Sealer: testSealer{}, Driver: driver}
	proposalDeps := proposal.Deps{
		Proposals: s, Rehearsals: s, Targets: s, Journal: s, Sealer: testSealer{},
		Driver: driver, Timeout: time.Second,
	}
	readDeps := read.Deps{Targets: s, Sealer: testSealer{}, Driver: driver, Timeout: time.Second}

	return New(Deps{
		Log: log,
		Targets: TargetUseCases{
			Create: target.NewCreate(targetDeps), List: target.NewList(targetDeps),
			Get: target.NewGet(targetDeps), Update: target.NewUpdate(targetDeps),
			Archive: target.NewArchive(targetDeps), Probe: target.NewProbe(targetDeps),
		},
		Read: ReadUseCases{
			Query: read.NewQuery(readDeps), Databases: read.NewDatabases(readDeps),
			Tables: read.NewTables(readDeps), Table: read.NewTable(readDeps),
		},
		Proposals: ProposalUseCases{
			Create: proposal.NewCreate(proposalDeps), List: proposal.NewList(proposalDeps),
			Get: proposal.NewGet(proposalDeps), Rehearse: proposal.NewRehearse(proposalDeps),
			Decide: proposal.NewDecide(proposalDeps), Apply: proposal.NewApply(proposalDeps),
			Events: proposal.NewEvents(proposalDeps),
		},
	})
}

// request runs one handler with a caller attached, the way the auth middleware would.
func request(t *testing.T, handler http.HandlerFunc, method, path, body string, caller domain.Caller) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req = req.WithContext(middlewares.WithCaller(req.Context(), caller))
	if ref := lastSegment(path); ref != "" {
		req.SetPathValue("ref", ref)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// lastSegment is the reference the mux would have put in the path value: the tests call the handler
// directly, so the path is scanned for the one segment that is a UUIDv7.
func lastSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		segment := parts[i]
		if len(segment) == 36 && strings.Count(segment, "-") == 4 {
			return segment
		}
	}
	return ""
}

var ops = domain.Caller{Name: "ops", Scopes: []string{"read", "admin", "approve", "apply"}}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("response is not the envelope: %v (%s)", err, rec.Body)
	}
	if out != nil && envelope.Data != nil {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			t.Fatalf("data: %v (%s)", err, envelope.Data)
		}
	}
}

// The health route is public and says one thing: the process is up.
func TestHealthz(t *testing.T) {
	h := newHandlers(newStore())
	rec := request(t, h.Healthz, http.MethodGet, "/healthz", "", domain.Anonymous)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	decodeBody(t, rec, &body)
	if body.Status != "ok" || body.Version != constants.Version {
		t.Fatalf("body = %+v", body)
	}
}

// Whoami answers the caller from the context, and never an empty array when it holds no scopes.
func TestWhoami(t *testing.T) {
	h := newHandlers(newStore())
	rec := request(t, h.Whoami, http.MethodGet, "/v1/whoami", "", domain.Caller{Name: "agent"})
	var body struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	decodeBody(t, rec, &body)
	if body.Name != "agent" || body.Scopes == nil || len(body.Scopes) != 0 {
		t.Fatalf("body = %+v", body)
	}
}

// A registration answers 201 with the target on the wire and no password anywhere in the body: the
// sealed value is not a field, so it cannot be leaked by a later refactor.
func TestCreateTargetNeverEchoesThePassword(t *testing.T) {
	h := newHandlers(newStore())
	rec := request(t, h.CreateTarget, http.MethodPost, "/v1/targets",
		`{"name":"staging","driver":"mysql","host":"db","database":"app","username":"noxis","password":"hunter2"}`,
		ops)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "hunter2") || strings.Contains(rec.Body.String(), "sealed:") {
		t.Fatalf("the response echoed the credential: %s", rec.Body)
	}
	var body struct {
		HasPassword bool `json:"has_password"`
		Port        int  `json:"port"`
	}
	decodeBody(t, rec, &body)
	if !body.HasPassword || body.Port != 3306 {
		t.Fatalf("body = %+v", body)
	}
}

// A body that is not JSON, or that carries a field nobody declared, is a 400 with the code the
// client branches on.
func TestCreateTargetRejectsABadBody(t *testing.T) {
	h := newHandlers(newStore())
	for _, body := range []string{"", "{", `{"name":"x","password":"y","bogus":1}`} {
		rec := request(t, h.CreateTarget, http.MethodPost, "/v1/targets", body, ops)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q answered %d, want 400", body, rec.Code)
		}
	}
}

func TestTargetNotFoundAndBadReference(t *testing.T) {
	h := newHandlers(newStore())

	rec := request(t, h.GetTarget, http.MethodGet, "/v1/targets/0192f0c0-0000-7000-8000-0000000000ff", "", ops)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown reference answered %d, want 404", rec.Code)
	}

	rec = request(t, h.GetTarget, http.MethodGet, "/v1/targets/prod", "", ops)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a name instead of a reference answered %d, want 400", rec.Code)
	}
}

func TestListTargets(t *testing.T) {
	s := newStore()
	s.seedTarget(targetRef)
	s.seedTarget("0192f0c0-0000-7000-8000-000000000003")

	rec := request(t, newHandlers(s).ListTargets, http.MethodGet, "/v1/targets", "", ops)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []struct {
		Ref string `json:"ref"`
	}
	decodeBody(t, rec, &body)
	if len(body) != 2 {
		t.Fatalf("listed %d targets, want 2", len(body))
	}
}

// The apply route answers 202 when the table is too large to wait for, with the proposal the caller
// is meant to poll.
func TestApplyAnswers202ForALargeTable(t *testing.T) {
	s := newStore()
	s.seedTarget(targetRef)
	s.seedProposal(proposalRef, targetRef, domain.StatusApproved)
	s.plan = domain.Plan{RowsEstimated: 2_000_000}

	rec := request(t, newHandlers(s).ApplyProposal, http.MethodPost, "/v1/proposals/"+proposalRef+"/apply", "", ops)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", rec.Code, rec.Body)
	}
	var body struct {
		Status string `json:"status"`
	}
	decodeBody(t, rec, &body)
	if body.Status != string(domain.StatusApplying) {
		t.Fatalf("status = %q, want applying", body.Status)
	}
}

// Applying something nobody approved is a conflict, and the transport says so with a code a client
// can branch on rather than a message it has to parse.
func TestApplyRefusesAnUnapprovedProposal(t *testing.T) {
	s := newStore()
	s.seedTarget(targetRef)
	s.seedProposal(proposalRef, targetRef, domain.StatusVerified)

	rec := request(t, newHandlers(s).ApplyProposal, http.MethodPost, "/v1/proposals/"+proposalRef+"/apply", "", ops)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", rec.Code, rec.Body)
	}
}

// A proposal is created from a body, rehearsed, and answered 201 with its evidence.
func TestCreateProposal(t *testing.T) {
	s := newStore()
	s.seedTarget(targetRef)
	s.rehearsal = domain.Rehearsal{Status: domain.RehearsalVerified, Note: "shadow"}

	rec := request(t, newHandlers(s).CreateProposal, http.MethodPost, "/v1/targets/"+targetRef+"/proposals",
		`{"title":"add a column","query":"ALTER TABLE users ADD COLUMN note TEXT"}`, ops)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body)
	}
	var body struct {
		Status string `json:"status"`
		Kind   string `json:"kind"`
	}
	decodeBody(t, rec, &body)
	if body.Status != string(domain.StatusVerified) || body.Kind != string(domain.KindDDL) {
		t.Fatalf("body = %+v", body)
	}
}

// The read route refuses a write before the driver is asked, and the refusal names the statement.
func TestRunQueryRefusesAWrite(t *testing.T) {
	s := newStore()
	s.seedTarget(targetRef)

	rec := request(t, newHandlers(s).RunQuery, http.MethodPost, "/v1/targets/"+targetRef+"/query",
		`{"sql":"DELETE FROM users"}`, ops)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body)
	}
}

// Every route answers something a client can act on. A 500 here means a handler reached a store it
// was not given, or panicked on a value it did not expect — the two failures a route table hides.
func TestEveryRouteAnswers(t *testing.T) {
	s := newStore()
	target := s.seedTarget(targetRef)
	s.seedProposal(proposalRef, targetRef, domain.StatusVerified)
	s.rehearsal = domain.Rehearsal{Status: domain.RehearsalVerified}
	s.driverCaps = domain.Capabilities{Version: "8.4.2"}
	h := newHandlers(s)

	host := "db2"
	password := "rotated"
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    string
	}{
		{"list targets", h.ListTargets, http.MethodGet, "/v1/targets", ""},
		{"get target", h.GetTarget, http.MethodGet, "/v1/targets/" + target.Ref, ""},
		{
			"update target", h.UpdateTarget, http.MethodPatch, "/v1/targets/" + target.Ref,
			`{"host":"` + host + `"}`,
		},
		{
			"rotate password", h.UpdateTarget, http.MethodPatch, "/v1/targets/" + target.Ref,
			`{"password":"` + password + `"}`,
		},
		{"probe target", h.ProbeTarget, http.MethodPost, "/v1/targets/" + target.Ref + "/probe", ""},
		{"capabilities", h.TargetCapabilities, http.MethodGet, "/v1/targets/" + target.Ref + "/capabilities", ""},
		{"query", h.RunQuery, http.MethodPost, "/v1/targets/" + target.Ref + "/query", `{"sql":"SELECT 1"}`},
		{"databases", h.ListDatabases, http.MethodGet, "/v1/targets/" + target.Ref + "/databases", ""},
		{"tables", h.ListTables, http.MethodGet, "/v1/targets/" + target.Ref + "/databases/app/tables", ""},
		{"table", h.ShowTable, http.MethodGet, "/v1/targets/" + target.Ref + "/databases/app/tables/users", ""},
		{"list proposals", h.ListProposals, http.MethodGet, "/v1/proposals", ""},
		{"get proposal", h.GetProposal, http.MethodGet, "/v1/proposals/" + proposalRef, ""},
		{"rehearse", h.RehearseProposal, http.MethodPost, "/v1/proposals/" + proposalRef + "/rehearse", ""},
		{"approve", h.ApproveProposal, http.MethodPost, "/v1/proposals/" + proposalRef + "/approve", `{}`},
		{"reject", h.RejectProposal, http.MethodPost, "/v1/proposals/" + proposalRef + "/reject", `{}`},
		{"events", h.ProposalEvents, http.MethodGet, "/v1/proposals/" + proposalRef + "/events", ""},
		{"archive", h.ArchiveTarget, http.MethodDelete, "/v1/targets/" + target.Ref, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := request(t, tc.handler, tc.method, tc.path, tc.body, ops)
			if rec.Code >= http.StatusInternalServerError {
				t.Fatalf("answered %d: %s", rec.Code, rec.Body)
			}
			if rec.Code == 0 {
				t.Fatal("the handler wrote no status")
			}
		})
	}
	// The sealer is a double, so the assertion is the shape: the row holds an envelope, not the
	// plaintext the caller typed.
	if stored := s.targets[targetRef].Secret; stored == password || !strings.HasPrefix(stored, "sealed:") {
		t.Fatalf("the rotated password was stored as %q", stored)
	}
}
