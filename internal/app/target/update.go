package target

import (
	"context"
	"fmt"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"time"
)

// Update changes what a target says about itself: its name, its user, its password, its
// TLS mode and its driver options. Host, port and database are absent on purpose — those
// are the identity of the server, and changing one makes a different target.
type Update struct{ Deps }

// NewUpdate builds the use case.
func NewUpdate(d Deps) *Update { return &Update{Deps: d} }

// Patch carries only the fields a caller sent. An empty password means "clear it"; nil
// means "leave it alone".
type Patch struct {
	Name     *string
	Username *string
	Password *string
	TLSMode  *string
	Options  *map[string]string
}

// UpdateInput is one patch and who sent it.
type UpdateInput struct {
	Ref   string
	Patch Patch
	Actor string
}

// Run reads, applies, validates and writes back. Validation runs after the patch, so a
// patch that would make the target invalid is refused rather than stored.
func (u *Update) Run(ctx context.Context, in UpdateInput) (domain.Target, error) {
	ref, err := requireRef(in.Ref)
	if err != nil {
		return domain.Target{}, err
	}
	target, err := u.Targets.ByRef(ctx, ref)
	if err != nil {
		return domain.Target{}, err
	}
	if target.Archived() {
		return domain.Target{}, fmt.Errorf("%w: target %q is archived", domain.ErrConflict, target.Name)
	}

	changed := map[string]string{}
	if in.Patch.Name != nil {
		target.Name = *in.Patch.Name
		changed["name"] = target.Name
	}
	if in.Patch.Username != nil {
		target.Username = *in.Patch.Username
		changed["username"] = target.Username
	}
	if in.Patch.TLSMode != nil {
		target.TLSMode = *in.Patch.TLSMode
		changed["tls_mode"] = target.TLSMode
	}
	if in.Patch.Options != nil {
		target.Options = *in.Patch.Options
		changed["options"] = "replaced"
	}
	if in.Patch.Password != nil {
		sealed, err := u.seal(target.Ref, *in.Patch.Password)
		if err != nil {
			return domain.Target{}, err
		}
		target.Secret = sealed
		// The new value is never named, only the fact that it moved.
		changed["password"] = "rotated"
	}
	if len(changed) == 0 {
		return domain.Target{}, fmt.Errorf("%w: the patch changes nothing", domain.ErrInvalid)
	}
	target.UpdatedAt = time.Now().UTC()
	if err := target.Validate(); err != nil {
		return domain.Target{}, err
	}

	stored, err := u.Targets.Update(ctx, target)
	if err != nil {
		return domain.Target{}, err
	}
	if err := u.record(ctx, stored, constants.JournalUpdated, in.Actor, changed); err != nil {
		return domain.Target{}, err
	}
	return stored, nil
}
