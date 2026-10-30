package permission

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List lists a connection's permissions.
type List struct{ usecases.RegistryDeps }

// @Summary List a connection's permissions
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.PermissionList
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/permissions [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	need, err := adminLevel(ctx, entity.LevelAdmin)
	if err != nil {
		uc.Fail(ctx, w, err)
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
	permissions, err := uc.Repositories.Permissions.ListPermissions(ctx, r.PathValue("name"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.PermissionList{Permissions: permissions})
}
