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
	ContainerStatus(ctx context.Context, name string) (string, error)
	ContainerRun(ctx context.Context, opts docker.ContainerOptions) error
	ContainerStart(ctx context.Context, name string) error
	ContainerStop(ctx context.Context, name string) error
	ContainerRestart(ctx context.Context, name string) error
	ContainerKill(ctx context.Context, name string) error
	ContainerRemove(ctx context.Context, name string) error
	ImageExists(ctx context.Context, image string) (bool, error)
	ImagePull(ctx context.Context, image string) error
	ImageRemove(ctx context.Context, image string) error
	VolumeExists(ctx context.Context, name string) (bool, error)
	VolumeCreate(ctx context.Context, name string) error
	VolumeRemove(ctx context.Context, name string) error
}
