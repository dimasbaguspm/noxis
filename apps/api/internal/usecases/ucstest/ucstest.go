// Package ucstest holds the fakes the use-case packages share in their tests.
// It is test-only: production code never imports it.
package ucstest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

// Conns is an in-memory ConnectionRepository.
type Conns struct{ M map[string]entity.Connection }

// NewConns returns a store with one postgres connection, "demo".
func NewConns() *Conns {
	return &Conns{M: map[string]entity.Connection{
		"demo": {Name: "demo", Driver: constants.DriverPostgres, Host: "db", Port: 5432, User: "postgres", Database: "demo"},
	}}
}

func (f *Conns) List(context.Context) ([]entity.Connection, error) {
	out := make([]entity.Connection, 0, len(f.M))
	for name := range f.M {
		out = append(out, f.M[name])
	}
	return out, nil
}

func (f *Conns) Get(_ context.Context, name string) (*entity.Connection, error) {
	c, ok := f.M[name]
	if !ok {
		return nil, constants.ErrNotFound
	}
	return &c, nil
}

func (f *Conns) GetByID(_ context.Context, id string) (*entity.Connection, error) {
	for name := range f.M {
		c := f.M[name]
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, constants.ErrNotFound
}

func (f *Conns) Save(_ context.Context, c entity.Connection) error {
	f.M[c.Name] = c
	return nil
}

func (f *Conns) Delete(_ context.Context, name string) error {
	delete(f.M, name)
	return nil
}

// Changes is an in-memory ChangeRepository.
type Changes struct{ M map[string]entity.Change }

func NewChanges() *Changes { return &Changes{M: map[string]entity.Change{}} }

func (f *Changes) ListChanges(context.Context) ([]entity.Change, error) {
	out := make([]entity.Change, 0, len(f.M))
	for id := range f.M {
		out = append(out, f.M[id])
	}
	return out, nil
}

func (f *Changes) GetChange(_ context.Context, id string) (*entity.Change, error) {
	c, ok := f.M[id]
	if !ok {
		return nil, constants.ErrNotFound
	}
	return &c, nil
}

func (f *Changes) SaveChange(_ context.Context, c entity.Change) error {
	f.M[c.ID] = c
	return nil
}

func (f *Changes) DeleteChange(_ context.Context, id string) error {
	delete(f.M, id)
	return nil
}

// Permissions is an in-memory PermissionRepository.
type Permissions struct{ M map[string]entity.Permission }

func NewPermissions() *Permissions {
	return &Permissions{M: map[string]entity.Permission{}}
}

func (f *Permissions) ListPermissions(context.Context, string) ([]entity.Permission, error) {
	out := make([]entity.Permission, 0, len(f.M))
	for _, p := range f.M {
		out = append(out, p)
	}
	return out, nil
}

func (f *Permissions) ListAllPermissions(context.Context) ([]entity.Permission, error) {
	out := make([]entity.Permission, 0, len(f.M))
	for _, p := range f.M {
		out = append(out, p)
	}
	return out, nil
}

func (f *Permissions) SetPermission(_ context.Context, p entity.Permission) error {
	f.M[p.Member] = p
	return nil
}

func (f *Permissions) DeletePermission(_ context.Context, _, member, _ string) error {
	delete(f.M, member)
	return nil
}

func (f *Permissions) ListPermissionsForMembers(_ context.Context, members []string) ([]entity.Permission, error) {
	want := map[string]struct{}{}
	for _, m := range members {
		want[m] = struct{}{}
	}
	out := make([]entity.Permission, 0)
	for _, p := range f.M {
		if _, ok := want[p.Member]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *Permissions) ListGroupNames(context.Context) ([]string, error) {
	seen := map[string]struct{}{}
	for _, p := range f.M {
		if p.MemberType == entity.MemberGroup {
			seen[p.Member] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for g := range seen {
		out = append(out, g)
	}
	return out, nil
}

// Users is an in-memory UserRepository.
type Users struct{ Accounts []entity.User }

func (f *Users) ListUsers(context.Context) ([]entity.User, error) { return f.Accounts, nil }
func (f *Users) CountUsers(context.Context) (int, error)          { return len(f.Accounts), nil }

func (f *Users) find(pred func(*entity.User) bool) (*entity.User, error) {
	for i := range f.Accounts {
		if pred(&f.Accounts[i]) {
			u := f.Accounts[i]
			return &u, nil
		}
	}
	return nil, constants.ErrNotFound
}

func (f *Users) GetUser(_ context.Context, subject string) (*entity.User, error) {
	return f.find(func(u *entity.User) bool { return u.Subject == subject })
}

func (f *Users) GetUserByLogin(_ context.Context, login string) (*entity.User, error) {
	if login == "" {
		return nil, constants.ErrNotFound
	}
	return f.find(func(u *entity.User) bool {
		return u.Password != "" && (u.Email == login || (u.Username != "" && u.Username == login))
	})
}

func (f *Users) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	if email == "" {
		return nil, constants.ErrNotFound
	}
	return f.find(func(u *entity.User) bool { return u.Email == email })
}

func (f *Users) EnsureSubject(ctx context.Context, subject, email string) error {
	if _, err := f.GetUser(ctx, subject); err == nil {
		return nil
	}
	f.Accounts = append(f.Accounts, entity.User{Subject: subject, Email: email, Role: entity.RoleMember, Status: entity.StatusActive})
	return nil
}

func (f *Users) SaveUser(_ context.Context, u entity.User) error {
	for i := range f.Accounts {
		if f.Accounts[i].Subject == u.Subject {
			f.Accounts[i] = u
			return nil
		}
	}
	f.Accounts = append(f.Accounts, u)
	return nil
}

func (f *Users) SetUserPassword(context.Context, string, string) error { return nil }

func (f *Users) SetUserStatus(ctx context.Context, subject, status string) error {
	u, err := f.GetUser(ctx, subject)
	if err != nil {
		return err
	}
	u.Status = status
	return f.SaveUser(ctx, *u)
}

func (f *Users) SetUserRole(ctx context.Context, subject, role string) error {
	u, err := f.GetUser(ctx, subject)
	if err != nil {
		return err
	}
	u.Role = role
	return f.SaveUser(ctx, *u)
}

func (f *Users) SetUserToken(context.Context, string, string, int64) error { return nil }

func (f *Users) SetLastLogin(ctx context.Context, subject string, at int64) error {
	u, err := f.GetUser(ctx, subject)
	if err != nil {
		return err
	}
	u.LastLoginAt = at
	return f.SaveUser(ctx, *u)
}

func (f *Users) SetSoleSuper(_ context.Context, subject string) error {
	for i := range f.Accounts {
		switch {
		case f.Accounts[i].Subject == subject:
			f.Accounts[i].Role = entity.RoleSuper
		case f.Accounts[i].Role == entity.RoleSuper:
			f.Accounts[i].Role = entity.RoleMember
		}
	}
	return nil
}

func (f *Users) ClearUserToken(context.Context, string) error { return nil }

func (f *Users) UserByToken(context.Context, string, int64) (*entity.User, error) {
	return nil, constants.ErrNotFound
}

// Events is an in-memory AccessEventRepository.
type Events struct{ List []entity.AccessEvent }

func (f *Events) ListAccessEvents(_ context.Context, subject string, limit int) ([]entity.AccessEvent, error) {
	out := make([]entity.AccessEvent, 0, len(f.List))
	for i := range f.List {
		e := f.List[i]
		if e.Subject == subject || e.Actor == subject {
			out = append(out, e)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// AccessRequests is an in-memory AccessRequestRepository.
type AccessRequests struct{ Rows []entity.AccessRequest }

func (f *AccessRequests) CreateAccessRequest(_ context.Context, r entity.AccessRequest) error {
	f.Rows = append(f.Rows, r)
	return nil
}

func (f *AccessRequests) PendingRequestExists(_ context.Context, subject, connection string) (bool, error) {
	for i := range f.Rows {
		if f.Rows[i].Subject == subject && f.Rows[i].Connection == connection && f.Rows[i].Status == entity.RequestPending {
			return true, nil
		}
	}
	return false, nil
}

func (f *AccessRequests) GetAccessRequest(_ context.Context, id string) (*entity.AccessRequest, error) {
	for i := range f.Rows {
		if f.Rows[i].ID == id {
			r := f.Rows[i]
			return &r, nil
		}
	}
	return nil, constants.ErrNotFound
}

func (f *AccessRequests) ListAccessRequests(_ context.Context, subject string) ([]entity.AccessRequest, error) {
	out := make([]entity.AccessRequest, 0)
	for i := range f.Rows {
		if f.Rows[i].Subject == subject {
			out = append(out, f.Rows[i])
		}
	}
	return out, nil
}

func (f *AccessRequests) ListPendingRequests(_ context.Context, connections []string) ([]entity.AccessRequest, error) {
	allowed := map[string]struct{}{}
	for _, c := range connections {
		allowed[c] = struct{}{}
	}
	out := make([]entity.AccessRequest, 0)
	for i := range f.Rows {
		if f.Rows[i].Status != entity.RequestPending {
			continue
		}
		if connections != nil {
			if _, ok := allowed[f.Rows[i].Connection]; !ok {
				continue
			}
		}
		out = append(out, f.Rows[i])
	}
	return out, nil
}

func (f *AccessRequests) SetAccessRequestDecision(_ context.Context, id, status, decidedBy string, decidedAt int64, note string) error {
	for i := range f.Rows {
		if f.Rows[i].ID == id {
			f.Rows[i].Status, f.Rows[i].DecidedBy, f.Rows[i].DecidedAt, f.Rows[i].DecisionNote = status, decidedBy, decidedAt, note
			return nil
		}
	}
	return constants.ErrNotFound
}

// SQL is a fake SQLModule with swappable failures.
type SQL struct {
	interfaces.SQLModule
	ValidateErr error
	ApplyErr    error
	WriteErr    error
	Ops         []string
}

func (f *SQL) Supports(driver string) bool {
	return driver == constants.DriverPostgres || driver == constants.DriverMySQL
}
func (f *SQL) Family() string           { return constants.FamilySQL }
func (f *SQL) VerifyWrite(string) error { return f.WriteErr }
func (f *SQL) Kind(string) string       { return constants.KindDDL }
func (f *SQL) ValidateChange(context.Context, entity.Connection, string, string) (interfaces.Result, error) {
	if f.ValidateErr != nil {
		return interfaces.Result{}, f.ValidateErr
	}
	return interfaces.Result{Message: "query and rollback both valid (rolled back)"}, nil
}
func (f *SQL) Apply(context.Context, entity.Connection, string) (interfaces.Result, error) {
	if f.ApplyErr != nil {
		return interfaces.Result{}, f.ApplyErr
	}
	return interfaces.Result{Message: "applied: 1 statement(s)"}, nil
}

// Memcache is a minimal module for the cross-family instance list.
type Memcache struct{ interfaces.Module }

func (Memcache) Family() string { return constants.FamilyMemcache }
func (Memcache) Supports(driver string) bool {
	return driver == constants.DriverRedis || driver == constants.DriverValkey
}

// Fixture wires the fakes into RegistryDeps.
type Fixture struct {
	Conns    *Conns
	Changes  *Changes
	Perms    *Permissions
	Users    *Users
	Events   *Events
	Requests *AccessRequests
	Mods     map[string]interfaces.Module
}

// New builds a fixture whose SQL family is sq.
func New(sq interfaces.SQLModule) *Fixture {
	return &Fixture{
		Conns:    NewConns(),
		Changes:  NewChanges(),
		Perms:    NewPermissions(),
		Users:    &Users{},
		Events:   &Events{},
		Requests: &AccessRequests{},
		Mods:     map[string]interfaces.Module{constants.FamilySQL: sq},
	}
}

// Deps returns the RegistryDeps every use case embeds.
func (f *Fixture) Deps() usecases.RegistryDeps {
	return usecases.RegistryDeps{
		Registry: repositories.Registry{
			Connections:    f.Conns,
			Changes:        f.Changes,
			Permissions:    f.Perms,
			Users:          f.Users,
			AccessEvents:   f.Events,
			AccessRequests: f.Requests,
		},
		Modules: modules.New(f.Mods, nil),
	}
}

// Mux registers one Serve on a fresh mux so the request path values (engine,
// name, id, table) are set the way the real router sets them.
func Mux(method, pattern string, h http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, h)
	return mux
}

// Do runs one request against a handler and returns the recorder. A non-nil
// principal is attached to the request context.
func Do(h http.Handler, method, path, body string, p *entity.Identity) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if p != nil {
		req = req.WithContext(usecases.WithPrincipal(req.Context(), *p))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
