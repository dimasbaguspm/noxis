package access

import (
	"context"
	"fmt"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

func canDecide(ctx context.Context, deps usecases.RegistryDeps, request *entity.AccessRequest, principal entity.Identity) error {
	if principal.Subject == request.Subject {
		return fmt.Errorf("%w: you cannot decide your own request", constants.ErrForbidden)
	}
	if principal.IsAdmin() {
		return nil
	}
	level, err := deps.EffectiveLevel(ctx, request.Connection, principal)
	if err != nil {
		return err
	}
	if !entity.LevelAllows(level, entity.LevelAdmin) {
		return fmt.Errorf("%w: deciding %s requires admin on %q", constants.ErrForbidden, request.ID, request.Connection)
	}
	return nil
}
