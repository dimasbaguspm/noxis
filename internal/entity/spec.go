// Package entity is the domain model. Persistent form lives here with yaml
// tags; the wire form lives in internal/core (proto). Converters at the core
// boundary map between the two — the domain never sees proto.
package entity

// Engine is one desired containerized engine on a node.
type Engine struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Version string            `yaml:"version"`
	Ports   map[string]string `yaml:"ports,omitempty"` // host -> container, overrides default
	Env     map[string]string `yaml:"env,omitempty"`   // overrides defaults
}

// StoredSpec is what the master persists per node: the assignment record.
type StoredSpec struct {
	Engines []Engine `yaml:"engines"`
}

// Spec is one node's assignment, in and out of the store.
type Spec struct {
	Node    string
	Engines []Engine
}
