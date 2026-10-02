// Package entity is the shared data model across store, wire and REST.
package entity

type Engine struct {
	Name    string            `json:"name" yaml:"name"`
	Type    string            `json:"type" yaml:"type"`
	Version string            `json:"version" yaml:"version"`
	Ports   map[string]string `json:"ports,omitempty" yaml:"ports,omitempty"`
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

type Spec struct {
	Node    string   `json:"node" yaml:"-"`
	Engines []Engine `json:"engines" yaml:"engines"`
}

// StoredSpec is the persisted form (the yaml body in sqlite).
type StoredSpec struct {
	Engines []Engine `json:"engines" yaml:"engines"`
}
