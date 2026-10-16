package usecases

import (
	"context"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ConnectionList is the "ConnectionList" use case.
type ConnectionList struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc ConnectionList) Serve(ctx context.Context, mod interfaces.Module) (out []entity.Connection, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	list, err := uc.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Connection, 0, len(list))
	for i := range list {
		c := &list[i]
		if !mod.Supports(c.Driver) {
			continue
		}
		ok, err := CanRead(ctx, uc.Permissions, c.Name)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, c.Redacted())
		}
	}
	return out, nil
}
