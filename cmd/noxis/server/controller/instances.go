package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// The instance routes are the cross-family view of a connection: any family's
// instance is addressable by its stable id, with the serving family resolved
// from the driver instead of named in the path.

// instance returns a connection decorated with the family that serves it, only
// when the caller may read it. A caller without read access gets not-found, so
// the id space never leaks an instance the caller cannot see.
func (d Deps) instance(ctx context.Context, id string) (out entity.Connection, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.Connections.GetByID(ctx, id)
	if err != nil {
		return entity.Connection{}, err
	}
	mod, ok := d.moduleByDriver(c.Driver)
	if !ok {
		return entity.Connection{}, interfaces.ErrUnsupported
	}
	readable, err := d.canRead(ctx, c.Name)
	if err != nil {
		return entity.Connection{}, err
	}
	if !readable {
		return entity.Connection{}, interfaces.ErrNotFound
	}
	c.Family = mod.Family()
	return c.Redacted(), nil
}

// instances lists every readable instance across families. Filters are
// optional: name and host match as substrings, driver matches exactly.
func (d Deps) instances(ctx context.Context, name, driver, host string) (out []entity.Connection, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(name)
	hostNeedle := strings.ToLower(host)
	out = make([]entity.Connection, 0, len(list))
	for i := range list {
		c := &list[i]
		mod, ok := d.moduleByDriver(c.Driver)
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
		readable, err := d.canRead(ctx, c.Name)
		if err != nil {
			return nil, err
		}
		if !readable {
			continue
		}
		c.Family = mod.Family()
		out = append(out, c.Redacted())
	}
	return out, nil
}

// @Summary List instances across engine families (passwords redacted)
// @Param name query string false "substring match on name"
// @Param driver query string false "exact driver match"
// @Param host query string false "substring match on host"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{instances=[]entity.Connection,total=int}
// @Router /v1/instances [get]
func ListInstances(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		q := r.URL.Query()
		instances, err := d.instances(ctx, q.Get("name"), q.Get("driver"), q.Get("host"))
		if err != nil {
			return nil, err
		}
		page, perPage := readPagination(r, 20)
		return map[string]any{
			"instances": pageSlice(instances, page, perPage),
			"total":     len(instances),
		}, nil
	})
}

// @Summary Rotate an instance's password (identity fields stay locked)
// @Param id path string true "instance id"
// @Param body body passwordBody true "new password"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /v1/instances/{id}/password [post]
func RotatePassword(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body passwordBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if err := d.rotatePassword(ctx, r.PathValue("id"), body.Password); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary Get one instance across engine families (password redacted)
// @Param id path string true "instance id"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/instances/{id} [get]
func GetInstance(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.instance(ctx, r.PathValue("id"))
	})
}

// rotatePassword replaces an instance's password. Identity fields are locked,
// so this is the only way to update a credential; it is a global-admin action.
func (d Deps) rotatePassword(ctx context.Context, id, password string) (err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if err := d.requireAdmin(ctx); err != nil {
		return err
	}
	if password == "" {
		return errors.New("password is required")
	}
	c, err := d.Connections.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if _, ok := d.moduleByDriver(c.Driver); !ok {
		return interfaces.ErrUnsupported
	}
	c.Password = password
	return d.Connections.Save(ctx, *c)
}

type passwordBody struct {
	Password string `json:"password"`
}
