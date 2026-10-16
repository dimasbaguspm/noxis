package handlers

import (
	"context"
	"net/http"

	"noxis/cmd/noxis/server/deps"
	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
)

// The instance routes are the cross-family view of a connection: any family's
// instance is addressable by its stable id, with the serving family resolved
// from the driver instead of named in the path.

// @Summary List instances across engine families (passwords redacted)
// @Param name query string false "substring match on name"
// @Param driver query string false "exact driver match"
// @Param host query string false "substring match on host"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{instances=[]entity.Connection,total=int}
// @Router /external/v1/instances [get]
func ListInstances(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		q := r.URL.Query()
		instances, err := d.Instances.Serve(ctx, q.Get("name"), q.Get("driver"), q.Get("host"))
		if err != nil {
			return nil, err
		}
		page, perPage := pagination.Params(r.URL.Query().Get("page"), r.URL.Query().Get("perPage"), constants.DefaultPageSize)
		return map[string]any{
			"instances": pagination.Slice(instances, page, perPage),
			"total":     len(instances),
		}, nil
	})
}

// @Summary Rotate an instance's password (identity fields stay locked)
// @Param id path string true "instance id"
// @Param body body passwordBody true "new password"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id}/password [post]
func RotatePassword(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		var body passwordBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if err := d.RotatePassword.Serve(ctx, r.PathValue("id"), body.Password); err != nil {
			return nil, err
		}
		return entity.StatusResponse{Status: "ok"}, nil
	})
}

// @Summary Get one instance across engine families (password redacted)
// @Param id path string true "instance id"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id} [get]
func GetInstance(w http.ResponseWriter, r *http.Request) {
	d := deps.From(r.Context())
	handle(w, r, func(ctx context.Context) (any, error) {
		return d.Instance.Serve(ctx, r.PathValue("id"))
	})
}

type passwordBody struct {
	Password string `json:"password"`
}
