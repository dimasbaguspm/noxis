package usecases

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// ConnectionLevels is the effective level per connection for the caller. A
// global admin holds admin everywhere; everyone else only appears with a
// matching grant, so an absent connection means no access.
type ConnectionLevels struct {
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc ConnectionLevels) Serve(ctx context.Context) (map[string]string, error) {
	p, ok := Principal(ctx)
	if !ok {
		return map[string]string{}, nil
	}
	list, err := uc.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(list))
	for i := range list {
		c := &list[i]
		if p.Admin {
			out[c.Name] = entity.LevelAdmin
			continue
		}
		level, err := EffectiveLevel(ctx, uc.Permissions, c.Name, p.Groups)
		if err != nil {
			return nil, err
		}
		if level != "" {
			out[c.Name] = level
		}
	}
	return out, nil
}
