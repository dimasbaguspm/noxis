package user

import (
	"net/http"
	"strings"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List returns accounts filtered by search and status.
type List struct{ usecases.RegistryDeps }

// @Summary List accounts (admin only)
// @Param q query string false "match subject, email or username"
// @Param status query string false "all, active or banned"
// @Success 200 {object} entity.UserList
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/users [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	users, err := uc.Users.ListUsers(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := r.URL.Query().Get("status")
	out := make([]entity.User, 0, len(users))
	for i := range users {
		user := &users[i]
		if query != "" && !matches(user, query) {
			continue
		}
		if status == entity.StatusActive && !user.Active() {
			continue
		}
		if status == entity.StatusBanned && user.Active() {
			continue
		}
		out = append(out, *user)
	}
	uc.Respond(w, entity.UserList{Users: out})
}
