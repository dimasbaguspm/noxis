package server

import (
	"context"
	"path/filepath"
	"testing"

	"noxis/internal/auth"
	"noxis/internal/common/config"
	"noxis/internal/repositories/sqlite"
)

// TestNewDepsWiresRepositories locks the composition root: a repository added to the Registry but not to sources would be a nil interface and only fail when a request reached it.
func TestNewDepsWiresRepositories(t *testing.T) {
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "noxis.db"), "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	d := newDeps(store, config.Config{}, auth.NewService(store, nil))
	for name, got := range map[string]any{
		"AccessRequests": d.AccessList.Repositories.AccessRequests,
		"AccessApprove":  d.AccessApprove.Repositories.AccessRequests,
		"AccessEvents":   d.UserGet.Repositories.AccessEvents,
		"Permissions":    d.SetPermission.Repositories.Permissions,
		"PermissionList": d.ListPermissions.Repositories.Permissions,
		"Users":          d.UserList.Repositories.Users,
		"Connections":    d.ConnectionList.Repositories.Connections,
		"Changes":        d.ChangeList.Repositories.Changes,
	} {
		if got == nil {
			t.Errorf("%s is not wired into the composition root", name)
		}
	}
}
