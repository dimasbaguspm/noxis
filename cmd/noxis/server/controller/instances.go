package controller

import (
	"context"
	"net/http"

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

// instanceChanges lists the changes raised for one instance. Grouping by kind
// and status is a console concern.
func (d Deps) instanceChanges(ctx context.Context, id string) (out []entity.Change, err error) {
	c, err := d.instance(ctx, id)
	if err != nil {
		return nil, err
	}
	all, err := d.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Connection == c.Name {
			out = append(out, all[i])
		}
	}
	return out, nil
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

// @Summary List the changes raised for one instance (newest first)
// @Param id path string true "instance id"
// @Success 200 {object} object{changes=[]entity.Change}
// @Failure 404 {object} map[string]string
// @Router /v1/instances/{id}/changes [get]
func ListInstanceChanges(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		changes, err := d.instanceChanges(ctx, r.PathValue("id"))
		return map[string]any{"changes": changes}, err
	})
}
