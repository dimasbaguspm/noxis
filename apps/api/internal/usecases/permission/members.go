package permission

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Members returns the known groups and user accounts for permission editing.
type Members struct{ usecases.RegistryDeps }

// @Summary The known groups and user accounts for permission editing
// @Success 200 {object} entity.MemberList
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/config/members [get]
func (uc Members) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	groups, err := uc.Repositories.Permissions.ListGroupNames(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	users, err := uc.Repositories.Users.ListUsers(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if groups == nil {
		groups = []string{}
	}
	pickable := make([]entity.Member, 0, len(users))
	for i := range users {
		user := &users[i]
		label := firstNonEmpty(user.Email, user.Username, user.Subject)
		pickable = append(pickable, entity.Member{Subject: user.Subject, Label: label, Role: user.Role})
	}
	uc.Respond(w, entity.MemberList{Groups: groups, Users: pickable})
}
