package apply

import (
	"context"
	"fmt"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

func changeFor(ctx context.Context, deps usecases.RegistryDeps, module interfaces.Module, id string) (*entity.Change, error) {
	switch module.(type) {
	case interfaces.SQLModule:
		return deps.SQLChange(ctx, id)
	case interfaces.MongoModule:
		return deps.MongoChange(ctx, id)
	}
	return nil, fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, module.Family())
}
