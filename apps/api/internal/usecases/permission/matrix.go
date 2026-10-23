package permission

import (
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// InstanceAccess is one instance with its whole access policy: who holds what,
// and until when. It is metadata only — a grant here says nothing about the
// caller's own reach.
type InstanceAccess struct {
	ID          string              `json:"id"`
	Connection  string              `json:"connection"`
	Driver      string              `json:"driver"`
	Family      string              `json:"family"`
	Permissions []entity.Permission `json:"permissions"`
}

// Matrix lists every instance with its grants, so an admin can see the access
// landscape per instance without opening each connection.
type Matrix struct{ usecases.RegistryDeps }

// @Summary Every instance and its access policy (admin only)
// @Success 200 {object} object{instances=[]InstanceAccess}
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/access [get]
func (uc Matrix) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(w, err)
		return
	}
	connections, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	grants, err := uc.Permissions.ListAllPermissions(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	byConnection := make(map[string][]entity.Permission, len(connections))
	for _, grant := range grants {
		byConnection[grant.Connection] = append(byConnection[grant.Connection], grant)
	}
	out := make([]InstanceAccess, 0, len(connections))
	for i := range connections {
		c := &connections[i]
		row := InstanceAccess{
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
	uc.Respond(w, map[string]any{"instances": out})
}
