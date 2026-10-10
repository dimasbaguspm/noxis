package usecases

import (
	"context"
	"fmt"
	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// DeleteGrant is the "DeleteGrant" use case.
type DeleteGrant struct{ Base }

// Serve runs it.
func (uc DeleteGrant) Serve(ctx context.Context, mod interfaces.Module, name, group string) (err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	if err := uc.RequireAdmin(ctx); err != nil {
		return err
	}
	if group == "" {
		return fmt.Errorf("group is required")
	}
	if _, err := uc.Raw(ctx, mod, name, entity.LevelRead); err != nil {
		return err
	}
	return uc.Grants.DeleteGrant(ctx, name, group)
}
