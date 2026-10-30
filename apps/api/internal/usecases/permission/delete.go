package permission

import (
	"fmt"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Delete removes a member's permission from a connection.
type Delete struct{ usecases.RegistryDeps }

// @Summary Revoke a member's permission on a connection
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param member query string true "user subject or group name"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/permissions [delete]
func (uc Delete) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	need, err := adminLevel(ctx, entity.LevelAdmin)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	member := r.URL.Query().Get("member")
	if member == "" {
		uc.Fail(ctx, w, fmt.Errorf("member is required"))
		return
	}
	module, err := uc.Modules.Engine(r.PathValue("driver"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, err := uc.RawConn(ctx, module, r.PathValue("name"), need); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Repositories.Permissions.DeletePermission(ctx, r.PathValue("name"), member, entity.Actor(ctx)); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.RespondNoContent(w)
}
