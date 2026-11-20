// Package read holds the use cases that look at a target without changing it: one statement, one
// schema listing, one table's definition. Each is its own file with one Run, and none calls
// another.
package read

import (
	"context"
	"fmt"
	"time"

	"noxis/internal/app/password"
	"noxis/internal/common/ref"
	"noxis/internal/domain"
	"noxis/internal/ports"
)

// Deps is what every read use case is built from.
type Deps struct {
	Targets ports.Targets
	Sealer  ports.Sealer
	Driver  ports.Driver
	// Timeout bounds one statement. It comes from QUERY_TIMEOUT and is applied here, so a route
	// cannot forget it.
	Timeout time.Duration
}

// connect loads the target behind a reference and opens its credential. A target that is
// archived is refused: a read would still work, but it would be a read of something the operator
// believes is put away.
func (d Deps) connect(ctx context.Context, value string) (domain.Target, string, error) {
	if !ref.Valid(value) {
		return domain.Target{}, "", fmt.Errorf("%w: %q is not a reference", domain.ErrInvalid, value)
	}
	target, err := d.Targets.ByRef(ctx, value)
	if err != nil {
		return domain.Target{}, "", err
	}
	if target.Archived() {
		return domain.Target{}, "", fmt.Errorf("%w: target %q is archived", domain.ErrConflict, target.Name)
	}
	plain, err := password.Open(d.Sealer, target)
	if err != nil {
		return domain.Target{}, "", err
	}
	return target, plain, nil
}
