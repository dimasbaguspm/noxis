package permission

import (
	"net/http"
	"time"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Set upserts one member's level on a connection.
type Set struct{ usecases.RegistryDeps }

// @Summary Grant a user or group a level on a connection (admin only)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param body body entity.PermissionGrant true "member, member type and level"
// @Success 200 {object} entity.Permission
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [put]
func (uc Set) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	module, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var body entity.PermissionGrant
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, err := uc.RawConn(ctx, module, r.PathValue("name"), ""); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	now := time.Now()
	permission := entity.Permission{
		Connection: r.PathValue("name"), Member: body.Member, MemberType: body.MemberType, Level: body.Level,
		ExpiresAt: now.Add(uc.GrantTTL()).Unix(),
		CreatedBy: entity.Actor(ctx),
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := permission.Validate(); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if err := uc.Permissions.SetPermission(ctx, permission); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, permission)
}
