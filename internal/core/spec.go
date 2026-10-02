package core

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SpecYAML is the human-editable form of desired state, stored on the master.
// The wire format is proto; YAML is only for storage and eyeballing.
type SpecYAML struct {
	Engines []EngineYAML `yaml:"engines"`
}

type EngineYAML struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Version string            `yaml:"version"`
	Ports   map[string]string `yaml:"ports,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

func SpecToYAML(spec []*EngineSpec) (string, error) {
	engines := make([]EngineYAML, 0, len(spec))
	for _, e := range spec {
		engines = append(engines, EngineYAML{
			Name: e.Name, Type: e.Type, Version: e.Version, Ports: e.Ports, Env: e.Env,
		})
	}
	b, err := yaml.Marshal(SpecYAML{Engines: engines})
	if err != nil {
		return "", fmt.Errorf("marshal spec: %w", err)
	}
	return string(b), nil
}

func SpecFromYAML(body string) ([]*EngineSpec, error) {
	var s SpecYAML
	if err := yaml.Unmarshal([]byte(body), &s); err != nil {
		return nil, fmt.Errorf("unmarshal spec: %w", err)
	}
	out := make([]*EngineSpec, 0, len(s.Engines))
	for _, e := range s.Engines {
		out = append(out, &EngineSpec{
			Name: e.Name, Type: e.Type, Version: e.Version, Ports: e.Ports, Env: e.Env,
		})
	}
	return out, nil
}