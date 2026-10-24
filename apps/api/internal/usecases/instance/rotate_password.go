package instance

import (
	"errors"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// RotatePassword rotates an instance's password.
type RotatePassword struct{ usecases.RegistryDeps }

// @Summary Rotate an instance's password (identity fields stay locked)
// @Param id path string true "instance id"
// @Param body body entity.InstancePassword true "new password"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id}/password [post]
func (uc RotatePassword) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	if err := uc.RequireSuper(ctx); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	var body entity.InstancePassword
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if body.Password == "" {
		uc.Fail(ctx, w, errors.New("password is required"))
		return
	}
	connection, err := uc.Repositories.Connections.GetByID(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	if _, ok := uc.Modules.Serves(connection.Driver); !ok {
		uc.Fail(ctx, w, constants.ErrUnsupported)
		return
	}
	connection.Password = body.Password
	if err := uc.Repositories.Connections.Save(ctx, *connection); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, entity.StatusResponse{Status: "ok"})
}
