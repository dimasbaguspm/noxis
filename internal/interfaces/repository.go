// Package interfaces is where the codebase agrees on its seams: everything a
// service needs from persistence. Implementations live in
// internal/repositories/{sqlite,kafka,slave,http}. Thin on purpose — the
// point is swap-ability, not a taxonomy.
package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// NodeRepository is the master's registry of registered slave nodes.
type NodeRepository interface {
	Register(ctx context.Context, id, name, ip string) error
	List(ctx context.Context) ([]entity.Slave, error)
}
