package usecases

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Members is the "Members" use case: the group names and user accounts a
// permission can target, for the console's member picker.
type Members struct {
	Permissions interfaces.PermissionRepository
	Users       interfaces.UserRepository
}

// Serve returns the known groups and every account.
func (uc Members) Serve(ctx context.Context) (groups []string, users []entity.User, err error) {
	if err := RequireAdmin(ctx); err != nil {
		return nil, nil, err
	}
	groups, err = uc.Permissions.ListGroupNames(ctx)
	if err != nil {
		return nil, nil, err
	}
	users, err = uc.Users.ListUsers(ctx)
	if groups == nil {
		groups = []string{}
	}
	if users == nil {
		users = []entity.User{}
	}
	return groups, users, err
}
