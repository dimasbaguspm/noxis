package target

import (
	"context"

	"noxis/internal/domain"
)

// Get reads one target by its reference. An unknown reference is ErrNotFound; a value
// that is not a reference at all is ErrInvalid, because those are different mistakes.
type Get struct{ Deps }

// NewGet builds the use case.
func NewGet(d Deps) *Get { return &Get{Deps: d} }

// Run reads the target.
func (g *Get) Run(ctx context.Context, ref string) (domain.Target, error) {
	value, err := requireRef(ref)
	if err != nil {
		return domain.Target{}, err
	}
	return g.Targets.ByRef(ctx, value)
}
