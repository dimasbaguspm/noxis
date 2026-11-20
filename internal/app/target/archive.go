package target

import (
	"context"
	"fmt"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"time"
)

// Archive puts a target away. A target is never deleted: the journal still points at it,
// and deleting the row would orphan the only record of what was done through it.
type Archive struct{ Deps }

// NewArchive builds the use case.
func NewArchive(d Deps) *Archive { return &Archive{Deps: d} }

// ArchiveInput names the target and the caller.
type ArchiveInput struct {
	Ref   string
	Actor string
}

// Run archives once. A second attempt is a conflict, not a no-op, because a caller that
// believes it archived something twice has lost track of the state.
func (a *Archive) Run(ctx context.Context, in ArchiveInput) (out0 domain.Target, err error) {
	ctx, end := telemetry.Span(ctx, "target.Archive.Run")
	defer func() { end(err) }()

	ref, err := requireRef(in.Ref)
	if err != nil {
		return domain.Target{}, err
	}
	target, err := a.Targets.ByRef(ctx, ref)
	if err != nil {
		return domain.Target{}, err
	}
	if target.Archived() {
		return domain.Target{}, fmt.Errorf("%w: target %q is already archived", domain.ErrConflict, target.Name)
	}

	now := time.Now().UTC()
	target.ArchivedAt = &now
	target.UpdatedAt = now
	stored, err := a.Targets.Update(ctx, target)
	if err != nil {
		return domain.Target{}, err
	}
	if err := a.record(ctx, stored, constants.JournalArchived, in.Actor, map[string]string{"name": stored.Name}); err != nil {
		return domain.Target{}, err
	}
	return stored, nil
}
