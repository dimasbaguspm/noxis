// Package runtimes holds everything that touches the machine: the docker
// driver and per-engine recipes. doorman's reconciler only decides; this
// package executes against the real runtime.
package runtimes

import (
	"encoding/json"
	"fmt"

	"noxis/internal/entity"
)

// namePrefix scopes engine containers to noxis on a shared docker host.
const namePrefix = "noxis-"

// desc is a stateless recipe for one engine type. Docker flattens every
// engine into "a container + ports + env", so this is all the abstraction
// amounts to.
//
// ponytail: Backup/Restore/Healthcheck arrive with the Phase 5 backup gate.
type desc struct {
	Type  string            // canonical type name
	Image string            // image name with %s for version, e.g. "postgres:%s"
	Ports map[string]string // default host -> container port bindings
	Env   map[string]string // default env vars (safe dev defaults)
}

var (
	registry = map[string]desc{}
	aliases  = map[string]string{}
)

func (d desc) image(version string) string {
	return fmt.Sprintf(d.Image, version)
}

func register(d desc, alt ...string) {
	registry[d.Type] = d
	for _, a := range alt {
		aliases[a] = d.Type
	}
}

// Lookup resolves a type name (canonical or alias) to its recipe.
func Lookup(typeName string) (desc, bool) {
	if canon, ok := aliases[typeName]; ok {
		typeName = canon
	}
	d, ok := registry[typeName]
	return d, ok
}

// ImageOf renders the pinned image for a desired engine.
func ImageOf(e entity.Engine) (string, bool) {
	d, ok := Lookup(e.Type)
	if !ok {
		return "", false
	}
	return fmt.Sprintf(d.Image, e.Version), true
}

// ContainerName names a node's engine container; the node prefix keeps
// sibling slaves from colliding on one docker host.
func ContainerName(node, engine string) string {
	return fmt.Sprintf("%s%s-%s", namePrefix, node, engine)
}

// DesiredLabel stamps the exact shape (version + merged ports/env) as a
// deterministic string. The reconciler compares it with the label on a live
// container: any drift — version bump, port change, env edit — means recreate.
func DesiredLabel(e entity.Engine) (string, error) {
	d, ok := Lookup(e.Type)
	if !ok {
		return "", fmt.Errorf("unknown engine type %q", e.Type)
	}
	b, err := json.Marshal(struct {
		Type    string            `json:"t"`
		Version string            `json:"v"`
		Ports   map[string]string `json:"p"`
		Env     map[string]string `json:"e"`
	}{e.Type, e.Version, mergedPorts(e, d), mergedEnv(e, d)})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DefaultContainerPort returns the container-side port for `NAME@VER:HOSTPORT`.
func DefaultContainerPort(typeName string) (string, bool) {
	d, ok := Lookup(typeName)
	if !ok || len(d.Ports) == 0 {
		return "", ok
	}
	for _, c := range d.Ports { // one default port per engine today
		return c, true
	}
	return "", true
}

func mergedPorts(e entity.Engine, d desc) map[string]string {
	if len(e.Ports) > 0 {
		return e.Ports
	}
	return d.Ports
}

func mergedEnv(e entity.Engine, d desc) map[string]string {
	out := map[string]string{}
	for k, v := range d.Env {
		out[k] = v
	}
	for k, v := range e.Env {
		out[k] = v
	}
	return out
}
