package rpc

import "noxis/internal/entity"

// Boundary maps: the domain model (entity) rides the wire as proto (this
// package). Conversions live here so the rest of the codebase never imports
// proto directly.

// SpecFromProto is the exported wire->domain boundary (slave consumes specs).
func SpecFromProto(p *DesiredSpec) *entity.Spec {
	s := &entity.Spec{}
	if p == nil {
		return s
	}
	for _, e := range p.Engines {
		s.Engines = append(s.Engines, entity.Engine{
			Name: e.Name, Type: e.Type, Version: e.Version,
			Ports: e.Ports, Env: e.Env,
		})
	}
	return s
}

// SpecToProto is the exported domain->wire boundary (CLI builds entity specs).
func SpecToProto(s *entity.Spec) *DesiredSpec {
	out := &DesiredSpec{}
	if s == nil {
		return out
	}
	for _, e := range s.Engines {
		out.Engines = append(out.Engines, &EngineSpec{
			Name: e.Name, Type: e.Type, Version: e.Version,
			Ports: e.Ports, Env: e.Env,
		})
	}
	return out
}

func statusFromProto(p *Status) entity.NodeStatus {
	return entity.NodeStatus{Node: p.Node, State: p.State, Detail: p.Detail}
}

func statusToProto(s entity.NodeStatus) *Status {
	return &Status{Node: s.Node, State: s.State, Detail: s.Detail, Ts: s.TS}
}
