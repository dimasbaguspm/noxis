// Package doorman manages database engines as pinned containers on the node
// it runs on. All brains live on the master; a doorman instance is a dumb
// executor that converges its node's slice of desired state.
package doorman

import "fmt"

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

func register(d desc, alt ...string) {
	registry[d.Type] = d
	for _, a := range alt {
		aliases[a] = d.Type
	}
}

// Lookup resolves a type name (canonical or alias) and reports whether it is
// a known engine.
func Lookup(typeName string) (desc, bool) {
	if canon, ok := aliases[typeName]; ok {
		typeName = canon
	}
	d, ok := registry[typeName]
	return d, ok
}

func (d desc) image(version string) string { return fmt.Sprintf(d.Image, version) }