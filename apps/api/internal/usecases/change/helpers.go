package change

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

func supports(module interfaces.Module) bool {
	switch module.(type) {
	case interfaces.SQLChangeEngine, interfaces.MongoChangeEngine, interfaces.SearchEngine:
		return true
	}
	return false
}

// readable keeps the changes the caller may see and the driver in the path actually serves: the connection decides the
// driver, so a mysql change never shows up under the postgres queue.
func readable(all []entity.Change, module interfaces.Module, allowed map[string]bool, drivers map[string]string, driver string) []entity.Change {
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		change := all[i]
		if !allowed[change.Connection] || drivers[change.Connection] != driver {
			continue
		}
		switch module.(type) {
		case interfaces.SQLChangeEngine:
			if change.Kind != constants.KindDML && change.Kind != constants.KindDDL {
				continue
			}
		case interfaces.MongoChangeEngine:
			// every document engine reviews the same shape of change, so a queue shows what its own engine wrote
			if change.Kind != constants.KindMongo && change.Kind != constants.KindDocument {
				continue
			}
		case interfaces.SearchEngine:
			if change.Kind != constants.KindSearch {
				continue
			}
		}
		change.Driver = driver
		out = append(out, change)
	}
	return out
}

// driverOf maps every registered connection to its driver, so a change can be scoped by the driver the route names.
func driverOf(ctx context.Context, deps usecases.RegistryDeps) (map[string]string, error) {
	all, err := deps.Repositories.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(all))
	for i := range all {
		out[all[i].Name] = all[i].Driver
	}
	return out, nil
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

// changeKind validates the queue's kind filter: an empty value keeps the queue whole, and only a SQL change engine serves
// the two SQL kinds.
func changeKind(raw string, module interfaces.Module) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if raw != constants.KindDML && raw != constants.KindDDL {
		return "", fmt.Errorf("kind must be %s or %s", constants.KindDML, constants.KindDDL)
	}
	if _, ok := module.(interfaces.SQLChangeEngine); !ok {
		return "", fmt.Errorf("%w: engine %q has no %s or %s kind", constants.ErrUnsupported, module.Driver(), constants.KindDML, constants.KindDDL)
	}
	return raw, nil
}

func filterByKind(all []entity.Change, kind string) []entity.Change {
	if kind == "" {
		return all
	}
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Kind == kind {
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
	case interfaces.SQLChangeEngine:
		return deps.SQLChange(ctx, id)
	case interfaces.MongoChangeEngine:
		return deps.MongoChange(ctx, id)
	case interfaces.SearchEngine:
		return deps.SearchChange(ctx, id)
	}
	return nil, unsupportedChangeFlow(module)
}

func decide(ctx context.Context, deps usecases.RegistryDeps, w http.ResponseWriter, r *http.Request, apply func(*entity.Change) error) {
	module, err := deps.Modules.Engine(r.PathValue("driver"))
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

// filterByText keeps the changes whose notes or statement mention the needle: the queue's name search is a reader's
// shorthand, not a SQL parser.
func filterByText(all []entity.Change, needle string) []entity.Change {
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return all
	}
	out := make([]entity.Change, 0, len(all))
	for i := range all {
		if strings.Contains(strings.ToLower(all[i].Notes), needle) ||
			strings.Contains(strings.ToLower(all[i].Query), needle) {
			out = append(out, all[i])
		}
	}
	return out
}
