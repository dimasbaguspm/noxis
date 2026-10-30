package app

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"noxis/internal/auth"
	"noxis/internal/repositories"
	"noxis/internal/repositories/sqlite"
)

// TestDefaultBuildsEveryUseCase fails when a use case is added to App and left unbuilt: the field stays zero and only the request that reaches it notices.
func TestDefaultBuildsEveryUseCase(t *testing.T) {
	store := openStore(t)
	a := Default(Deps{Repositories: repositories.Default(store), AuthService: auth.NewService(store, nil)})
	value := reflect.ValueOf(a)
	checked := 0
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if _, ok := field.Type.MethodByName("Serve"); !ok {
			continue // Modules is a seam the routes read, not a use case
		}
		checked++
		if value.Field(i).IsZero() {
			t.Errorf("%s is not built by Default", field.Name)
		}
	}
	// A field renamed or a method moved off the value receiver would make the loop vacuous, which is worse than a failure.
	if checked == 0 {
		t.Fatal("no use case field carries a Serve method")
	}
}

// TestDefaultWiresRepositories locks the other half: a repository added to the Registry but not passed to the use cases is a nil interface that only fails when a request reaches it.
func TestDefaultWiresRepositories(t *testing.T) {
	store := openStore(t)
	a := Default(Deps{Repositories: repositories.Default(store), AuthService: auth.NewService(store, nil)})
	for name, got := range map[string]any{
		"AccessRequests": a.AccessList.Repositories.AccessRequests,
		"AccessApprove":  a.AccessApprove.Repositories.AccessRequests,
		"AccessEvents":   a.UserGet.Repositories.AccessEvents,
		"Permissions":    a.SetPermission.Repositories.Permissions,
		"PermissionList": a.ListPermissions.Repositories.Permissions,
		"Users":          a.UserList.Repositories.Users,
		"Connections":    a.ConnectionList.Repositories.Connections,
		"Changes":        a.ChangeList.Repositories.Changes,
	} {
		if got == nil {
			t.Errorf("%s is not wired into the composition root", name)
		}
	}
}

// openStore gives the composition root a real store, so the test fails on wiring rather than on a fake that hides a nil.
func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}
