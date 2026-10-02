package interfaces

import (
	"context"

	"noxis/internal/repositories/docker"
)

// Docker is the docker service capability: networks (the node's isolation
// boundary) and container lifecycle (run, inspect for drift/status, remove
// for cleanup, list). Implemented by internal/repositories/docker; feature
// code (doorman) decides against this seam and never builds shell args.
type Docker interface {
	NetworkExists(ctx context.Context, label string) (bool, error)
	NetworkCreate(ctx context.Context, name string, labels ...string) error
	NetworkRemove(ctx context.Context, name string) error
	ContainerList(ctx context.Context, label string) ([]string, error)
	ContainerInspect(ctx context.Context, name string) (docker.Container, error)
	ContainerRun(ctx context.Context, opts docker.ContainerOptions) error
	ContainerRemove(ctx context.Context, name string) error
}
