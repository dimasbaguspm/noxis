package permission

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Matrix lists every instance with its grants, so an admin can see the access
// landscape per instance without opening each connection.
type Matrix struct{ usecases.RegistryDeps }

// @Summary Every instance and its access policy (admin only)
// @Success 200 {object} entity.InstanceAccessList
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/access [get]
func (uc Matrix) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	connections, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	grants, err := uc.Permissions.ListAllPermissions(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	byConnection := make(map[string][]entity.Permission, len(connections))
	for _, grant := range grants {
		byConnection[grant.Connection] = append(byConnection[grant.Connection], grant)
	}
	out := make([]entity.InstanceAccess, 0, len(connections))
	for i := range connections {
		c := &connections[i]
		row := entity.InstanceAccess{
			ID: c.ID, Connection: c.Name, Driver: c.Driver,
			Permissions: byConnection[c.Name],
		}
		if mod, ok := uc.Modules.Serves(c.Driver); ok {
			row.Family = mod.Family()
		}
		if row.Permissions == nil {
			row.Permissions = []entity.Permission{}
		}
		out = append(out, row)
	}
	uc.Respond(w, entity.InstanceAccessList{Instances: out})
}
