package change

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/common"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

func supports(module interfaces.Module) bool {
	switch module.(type) {
	case interfaces.SQLModule, interfaces.MongoModule:
		return true
	}
	return false
}

func readable(all []entity.Change, module interfaces.Module, allowed map[string]bool) []entity.Change {
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		change := all[i]
		if !allowed[change.Connection] {
			continue
		}
		switch module.(type) {
		case interfaces.SQLModule:
			if change.Kind == constants.KindDML || change.Kind == constants.KindDDL {
				out = append(out, change)
			}
		case interfaces.MongoModule:
			if change.Kind == constants.KindMongo {
				out = append(out, change)
			}
		}
	}
	return out
}

func filterByConnection(all []entity.Change, needle string) []entity.Change {
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return all
	}
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		if strings.Contains(strings.ToLower(all[i].Connection), needle) {
			out = append(out, all[i])
		}
	}
	return out
}

func filterByStatus(all []entity.Change, status string) []entity.Change {
	status = strings.TrimSpace(status)
	if status == "" {
		return all
	}
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Status == status {
			out = append(out, all[i])
		}
	}
	return out
}

func unsupportedChangeFlow(module interfaces.Module) error {
	return fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, module.Driver())
}

func changeFor(ctx context.Context, deps usecases.RegistryDeps, module interfaces.Module, id string) (*entity.Change, error) {
	switch module.(type) {
	case interfaces.SQLModule:
		return deps.SQLChange(ctx, id)
	case interfaces.MongoModule:
		return deps.MongoChange(ctx, id)
	}
	return nil, unsupportedChangeFlow(module)
}

// flowFor is the change queue of the driver the route names, so the driver picks the queue rather than the change's kind.
func flowFor(ctx context.Context, driver string) (interfaces.ChangeFlow, error) {
	var flow interfaces.ChangeFlow
	if err := common.LoadModule(common.WithModuleDriver(ctx, driver), &flow); err != nil {
		return nil, err
	}
	return flow, nil
}

func decide(ctx context.Context, deps usecases.RegistryDeps, w http.ResponseWriter, r *http.Request, apply func(*entity.Change) error) {
	module, err := deps.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		deps.Fail(ctx, w, err)
		return
	}
	change, err := changeFor(ctx, deps, module, r.PathValue("id"))
	if err != nil {
		deps.Fail(ctx, w, err)
		return
	}
	if err := deps.Authorize(ctx, change.Connection, entity.LevelAdmin); err != nil {
		deps.Fail(ctx, w, err)
		return
	}
	if err := apply(change); err != nil {
		deps.Fail(ctx, w, err)
		return
	}
	deps.Respond(w, change)
}
