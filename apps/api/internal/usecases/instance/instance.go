// Package instance holds the cross-family instance use cases.
package instance

import (
	"errors"
	"net/http"
	"strings"

	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// List lists instances across engine families (passwords redacted).
type List struct{ usecases.RegistryDeps }

// @Summary List instances across engine families (passwords redacted)
// @Param name query string false "substring match on name"
// @Param driver query string false "exact driver match"
// @Param host query string false "substring match on host"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{instances=[]entity.Connection,total=int}
// @Router /external/v1/instances [get]
func (uc List) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	all, err := uc.Connections.List(ctx)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	needle := strings.ToLower(q.Get("name"))
	driver := q.Get("driver")
	hostNeedle := strings.ToLower(q.Get("host"))
	out := make([]entity.Connection, 0, len(all))
	for i := range all {
		c := &all[i]
		mod, ok := uc.Modules.Serves(c.Driver)
		if !ok {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(c.Name), needle) {
			continue
		}
		if driver != "" && !strings.EqualFold(c.Driver, driver) {
			continue
		}
		if hostNeedle != "" && !strings.Contains(strings.ToLower(c.Host), hostNeedle) {
			continue
		}
		readable, err := uc.CanRead(ctx, c.Name)
		if err != nil {
			uc.Fail(w, err)
			return
		}
		if !readable {
			continue
		}
		c.Family = mod.Family()
		out = append(out, c.Redacted())
	}
	page, perPage := pagination.Params(q.Get("page"), q.Get("perPage"), constants.DefaultPageSize)
	uc.Respond(w, map[string]any{
		"instances": pagination.Slice(out, page, perPage),
		"total":     len(out),
	})
}

// Get returns one instance across engine families (password redacted).
type Get struct{ usecases.RegistryDeps }

// @Summary Get one instance across engine families (password redacted)
// @Param id path string true "instance id"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id} [get]
func (uc Get) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, err := uc.Connections.GetByID(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	mod, ok := uc.Modules.Serves(c.Driver)
	if !ok {
		uc.Fail(w, constants.ErrUnsupported)
		return
	}
	readable, err := uc.CanRead(ctx, c.Name)
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if !readable {
		uc.Fail(w, constants.ErrNotFound)
		return
	}
	c.Family = mod.Family()
	uc.Respond(w, c.Redacted())
}

// passwordBody is the rotate-password payload.
type passwordBody struct {
	Password string `json:"password"`
}

// RotatePassword rotates an instance's password.
type RotatePassword struct{ usecases.RegistryDeps }

// @Summary Rotate an instance's password (identity fields stay locked)
// @Param id path string true "instance id"
// @Param body body passwordBody true "new password"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id}/password [post]
func (uc RotatePassword) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := uc.RequireAdmin(ctx); err != nil {
		uc.Fail(w, err)
		return
	}
	var body passwordBody
	if err := uc.Decode(r, &body); err != nil {
		uc.Fail(w, err)
		return
	}
	if body.Password == "" {
		uc.Fail(w, errors.New("password is required"))
		return
	}
	c, err := uc.Connections.GetByID(ctx, r.PathValue("id"))
	if err != nil {
		uc.Fail(w, err)
		return
	}
	if _, ok := uc.Modules.Serves(c.Driver); !ok {
		uc.Fail(w, constants.ErrUnsupported)
		return
	}
	c.Password = body.Password
	if err := uc.Connections.Save(ctx, *c); err != nil {
		uc.Fail(w, err)
		return
	}
	uc.Respond(w, entity.StatusResponse{Status: "ok"})
}
