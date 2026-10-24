package user

import (
	"context"
	"fmt"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

func matches(user *entity.User, query string) bool {
	return strings.Contains(strings.ToLower(user.Subject), query) ||
		strings.Contains(strings.ToLower(user.Email), query) ||
		strings.Contains(strings.ToLower(user.Username), query)
}

func (uc Update) applyRole(ctx context.Context, caller entity.Identity, user *entity.User, role *string) error {
	if role == nil {
		return nil
	}
	if !caller.IsSuper() {
		return fmt.Errorf("%w: only the super admin changes roles", constants.ErrForbidden)
	}
	if *role != entity.RoleAdmin && *role != entity.RoleMember {
		return fmt.Errorf("role must be %s or %s", entity.RoleAdmin, entity.RoleMember)
	}
	if *role == user.Role {
		return nil
	}
	if err := uc.Repositories.Users.SetUserRole(ctx, user.Subject, *role); err != nil {
		return err
	}
	user.Role = *role
	return nil
}

func (uc Update) applyStatus(ctx context.Context, caller entity.Identity, user *entity.User, status *string) error {
	if status == nil {
		return nil
	}
	if !caller.IsSuper() && user.IsAdmin() {
		return fmt.Errorf("%w: only the super admin changes another admin's status", constants.ErrForbidden)
	}
	if *status == user.Status {
		return nil
	}
	if err := uc.Repositories.Users.SetUserStatus(ctx, user.Subject, *status); err != nil {
		return err
	}
	user.Status = *status
	return nil
}

func (uc Update) checkPatch(caller entity.Identity, user *entity.User, body entity.UserUpdate) error {
	if body.Role != nil {
		if !caller.IsSuper() {
			return fmt.Errorf("%w: only the super admin changes roles", constants.ErrForbidden)
		}
		if *body.Role != entity.RoleAdmin && *body.Role != entity.RoleMember {
			return fmt.Errorf("role must be %s or %s", entity.RoleAdmin, entity.RoleMember)
		}
	}
	if body.Status != nil {
		if *body.Status != entity.StatusActive && *body.Status != entity.StatusBanned {
			return fmt.Errorf("status must be %s or %s", entity.StatusActive, entity.StatusBanned)
		}
		if !caller.IsSuper() && user.IsAdmin() {
			return fmt.Errorf("%w: only the super admin changes another admin's status", constants.ErrForbidden)
		}
	}
	return nil
}
