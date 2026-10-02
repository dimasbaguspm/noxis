package rpc

import "noxis/internal/entity"

// Domain model rides the stream as proto; conversions live here.

func SpecToProto(s *entity.Spec) *DesiredSpec {
	out := &DesiredSpec{}
	for _, e := range s.Engines {
		out.Engines = append(out.Engines, &EngineSpec{
			Name: e.Name, Type: e.Type, Version: e.Version, Ports: e.Ports, Env: e.Env,
		})
	}
	return out
}

func SpecFromProto(p *DesiredSpec) *entity.Spec {
	s := &entity.Spec{}
	for _, e := range p.Engines {
		s.Engines = append(s.Engines, entity.Engine{
			Name: e.Name, Type: e.Type, Version: e.Version, Ports: e.Ports, Env: e.Env,
		})
	}
	return s
}
