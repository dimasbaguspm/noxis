// Package interfaces is where the codebase agrees on its seams (one file per
// service) and their data shapes. Implementations live in
// internal/repositories/* and assert satisfiability right next to their
// structs. Thin on purpose — swap-ability in tests, not a taxonomy.
package interfaces

import (
	"context"
	"errors"
)

// ContainerOptions is everything `docker run` needs for one engine.
type ContainerOptions struct {
	Name         string
	Image        string
	Labels       map[string]string
	Env          map[string]string
	Ports        map[string]string // host:container, external key
	Volumes      []string          // name:path entries
	Network      string
	NetworkAlias string
}

// Container facts the feature decides on (drift checks, queries).
type (
	Container struct {
		State  ContainerState  `json:"State"`
		Config ContainerConfig `json:"Config"`
	}
	ContainerState struct {
		Running bool `json:"Running"`
	}
	ContainerConfig struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	}
)

// ErrNotFound is returned when a docker object does not exist.
var ErrNotFound = errors.New("not found")

// Docker is the docker service capability: networks (the node's isolation
// boundary, reconciled by NetworkReconcile) and container lifecycle (run,
// exec, inspect for drift/status, remove for cleanup, list). Implemented by
// internal/repositories/docker; feature code (doorman, the slave) decides
// against this seam and never builds shell args.
type Docker interface {
	NetworkExists(ctx context.Context, label string) (bool, error)
	NetworkCreate(ctx context.Context, name string, labels ...string) error
	NetworkRemove(ctx context.Context, name string) error
	NetworkReconcile(ctx context.Context, name, label string, engines int) (string, error)
	ContainerList(ctx context.Context, label string) ([]string, error)
	ContainerInspect(ctx context.Context, name string) (Container, error)
	ContainerStatus(ctx context.Context, name string) (string, error)
	ContainerRun(ctx context.Context, opts ContainerOptions) error
	ContainerExec(ctx context.Context, name string, args ...string) (string, error)
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
