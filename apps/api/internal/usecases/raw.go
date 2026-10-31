package usecases

import (
	"context"
	"fmt"
	"slices"

	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// RawConn checks the driver and access before returning a password-bearing connection.
func (d RegistryDeps) RawConn(ctx context.Context, mod interfaces.Module, name, level string) (*entity.Connection, error) {
	c, err := d.Repositories.Connections.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !mod.Supports(c.Driver) {
		logging.WarnWithCtx(ctx, "connection belongs to another driver",
			"connection", c.Name, "driver", c.Driver, "engine", mod.Driver())
		return nil, fmt.Errorf("connection %q is a %s connection, not %s", c.Name, c.Driver, mod.Driver())
	}
	if err := d.Authorize(ctx, name, level); err != nil {
		return nil, err
	}
	return c, nil
}

// Change loads a change and insists it carries one of the kinds this queue serves, so a request cannot be opened through
// another engine's flow. what names the queue in the refusal, because the reader cares which one turned them away.
func (d RegistryDeps) Change(ctx context.Context, id, what string, kinds ...string) (*entity.Change, error) {
	ch, err := d.Repositories.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(kinds, ch.Kind) {
		return nil, fmt.Errorf("change %s is %s, not a %s", id, ch.Kind, what)
	}
	if err := d.Authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// ChangeFor loads the change through the flow its engine owns: the module's capability picks the kinds and the wording.
func (d RegistryDeps) ChangeFor(ctx context.Context, module interfaces.Module, id string) (*entity.Change, error) {
	switch module.(type) {
	case interfaces.SQLChangeEngine:
		return d.Change(ctx, id, "sql change", constants.KindDML, constants.KindDDL)
	case interfaces.MongoChangeEngine:
		return d.Change(ctx, id, "document request", constants.KindMongo, constants.KindDocument)
	case interfaces.SearchEngine:
		return d.Change(ctx, id, "search request", constants.KindSearch)
	}
	return nil, UnsupportedChangeFlow(module)
}

// UnsupportedChangeFlow refuses an engine that reviews no changes at all, naming it rather than the queue it lacks.
func UnsupportedChangeFlow(module interfaces.Module) error {
	return fmt.Errorf("%w: engine %q does not serve changes", constants.ErrUnsupported, module.Driver())
}
