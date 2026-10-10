package usecases

import (
	"context"
	"errors"
	"strings"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// Instance returns a connection decorated with the family that serves it, only
// when the caller may read it. A caller without read access gets not-found, so
// the id space never leaks an instance the caller cannot see.
func (a *App) Instance(ctx context.Context, id string) (out entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := a.Connections.GetByID(ctx, id)
	if err != nil {
		return entity.Connection{}, err
	}
	mod, ok := a.moduleByDriver(c.Driver)
	if !ok {
		return entity.Connection{}, interfaces.ErrUnsupported
	}
	readable, err := a.CanRead(ctx, c.Name)
	if err != nil {
		return entity.Connection{}, err
	}
	if !readable {
		return entity.Connection{}, interfaces.ErrNotFound
	}
	c.Family = mod.Family()
	return c.Redacted(), nil
}

// Instances lists every readable instance across families. Filters are
// optional: name and host match as substrings, driver matches exactly.
func (a *App) Instances(ctx context.Context, name, driver, host string) (out []entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	list, err := a.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(name)
	hostNeedle := strings.ToLower(host)
	out = make([]entity.Connection, 0, len(list))
	for i := range list {
		c := &list[i]
		mod, ok := a.moduleByDriver(c.Driver)
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
		readable, err := a.CanRead(ctx, c.Name)
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

// RotatePassword replaces an instance's password. Identity fields are locked,
// so this is the only way to update a credential; it is a global-admin action.
func (a *App) RotatePassword(ctx context.Context, id, password string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := a.RequireAdmin(ctx); err != nil {
		return err
	}
	if password == "" {
		return errors.New("password is required")
	}
	c, err := a.Connections.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if _, ok := a.moduleByDriver(c.Driver); !ok {
		return interfaces.ErrUnsupported
	}
	c.Password = password
	return a.Connections.Save(ctx, *c)
}
