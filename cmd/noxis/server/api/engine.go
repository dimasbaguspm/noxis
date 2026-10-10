package api

import (
	"fmt"
	"net/http"

	"noxis/internal/interfaces"
)

// engine resolves the module named by the request's {engine} path parameter
// through the application's registry.
func (s *Server) engine(r *http.Request) (interfaces.Module, error) {
	name := r.PathValue("engine")
	if mod, ok := s.App.Modules[name]; ok {
		return mod, nil
	}
	return nil, fmt.Errorf("%w: no engine %q", interfaces.ErrUnsupported, name)
}

// sqlOf, mongoOf and memcacheOf narrow a module to the capability a handler
// needs; a module that does not serve it is unsupported (404).
func sqlOf(mod interfaces.Module) (interfaces.SQLModule, error) {
	m, ok := mod.(interfaces.SQLModule)
	if !ok {
		return nil, unsupported(mod, "sql operations")
	}
	return m, nil
}

func mongoOf(mod interfaces.Module) (interfaces.MongoModule, error) {
	m, ok := mod.(interfaces.MongoModule)
	if !ok {
		return nil, unsupported(mod, "mongodb operations")
	}
	return m, nil
}

func memcacheOf(mod interfaces.Module) (interfaces.MemcacheModule, error) {
	m, ok := mod.(interfaces.MemcacheModule)
	if !ok {
		return nil, unsupported(mod, "memory-cache operations")
	}
	return m, nil
}

func unsupported(mod interfaces.Module, what string) error {
	return fmt.Errorf("%w: engine %q does not serve %s", interfaces.ErrUnsupported, mod.Family(), what)
}

// engineSQL, engineMongo and engineMemcache resolve the request's engine and
// narrow it to the capability the handler serves.
func (s *Server) engineSQL(r *http.Request) (interfaces.SQLModule, error) {
	mod, err := s.engine(r)
	if err != nil {
		return nil, err
	}
	return sqlOf(mod)
}

func (s *Server) engineMongo(r *http.Request) (interfaces.MongoModule, error) {
	mod, err := s.engine(r)
	if err != nil {
		return nil, err
	}
	return mongoOf(mod)
}

func (s *Server) engineMemcache(r *http.Request) (interfaces.MemcacheModule, error) {
	mod, err := s.engine(r)
	if err != nil {
		return nil, err
	}
	return memcacheOf(mod)
}