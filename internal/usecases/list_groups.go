package usecases

import "context"

// Groups is the "Groups" use case.
type Groups struct{ Base }

// Serve runs it.
func (uc Groups) Serve(ctx context.Context) ([]string, error) {
	if err := uc.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return uc.Grants.ListGroupNames(ctx)
}
