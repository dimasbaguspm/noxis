// Package interfaces is where the codebase agrees on its seams: everything a
// service needs from persistence and notification. Implementations live in
// internal/repository. Thin on purpose — the point is swap-ability in tests,
// not a taxonomy.
package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// SpecRepository keeps the per-node assignment records.
type SpecRepository interface {
	Get(ctx context.Context, node string) (*entity.Spec, error)
	Set(ctx context.Context, spec *entity.Spec) error
	// Subscribe returns a coalescing channel: a tick means "spec changed, re-read".
	// The channel is owned by the caller until Unsubscribe.
	Subscribe(node string) chan struct{}
	Unsubscribe(node string, ch chan struct{})
}

// StatusRepository keeps the last reported runtime state per node.
type StatusRepository interface {
	Report(ctx context.Context, s entity.NodeStatus) error
	List(ctx context.Context) ([]entity.NodeStatus, error)
}