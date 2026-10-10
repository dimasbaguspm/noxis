package usecases

import (
	"context"
	"noxis/internal/interfaces"
)

// Groups is the "Groups" use case.
type Groups struct {
	Grants interfaces.GrantRepository
}

// Serve runs it.
func (uc Groups) Serve(ctx context.Context) ([]string, error) {
	if err := RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return uc.Grants.ListGroupNames(ctx)
}
