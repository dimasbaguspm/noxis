package auth

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Me returns the authenticated caller and its per-connection levels.
type Me struct{ usecases.RegistryDeps }

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} entity.MeResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/me [get]
func (uc Me) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	principal, ok := entity.Principal(ctx)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnauthorized)
		return
	}
	grants := map[string]entity.AccessGrant{}
	connections, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	for i := range connections {
		connection := &connections[i]
		if principal.IsSuper() {
			grants[connection.Name] = entity.AccessGrant{Level: entity.LevelAdmin}
			continue
		}
		level, expiresAt, err := uc.EffectiveGrant(ctx, connection.Name, principal)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		if level != "" {
			grants[connection.Name] = entity.AccessGrant{Level: level, ExpiresAt: expiresAt}
		}
	}
	uc.Respond(w, entity.MeResponse{Identity: principal, Connections: grants})
}
