package instance

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/usecases"
)

// Get returns one instance across engine families (password redacted).
type Get struct{ usecases.RegistryDeps }

// @Summary Get one instance across engine families (password redacted)
// @Param id path string true "instance id"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	connection, err := uc.Repositories.Connections.GetByID(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	module, ok := uc.Modules.Serves(connection.Driver)
	if !ok {
		uc.Fail(ctx, w, constants.ErrUnsupported)
		return
	}
	readable, err := uc.CanRead(ctx, connection.Name)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if !readable {
		uc.Fail(ctx, w, constants.ErrNotFound)
		return
	}
	connection.Family = module.Family()
	uc.Respond(w, connection.Redacted())
}
