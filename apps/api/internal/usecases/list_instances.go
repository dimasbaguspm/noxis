package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"strings"
)

// Instances is the "Instances" use case.
type ListInstances struct {
	Connections interfaces.ConnectionRepository
	Grants      interfaces.GrantRepository
}

// Serve runs it.
func (uc ListInstances) Serve(ctx context.Context, name, driver, host string) (out []entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	list, err := uc.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(name)
	hostNeedle := strings.ToLower(host)
	out = make([]entity.Connection, 0, len(list))
	for i := range list {
		c := &list[i]
		mod, ok := modules.RegistryFrom(ctx).Serves(c.Driver)
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
		readable, err := CanRead(ctx, uc.Grants, c.Name)
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
