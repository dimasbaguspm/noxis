package deps

import (
	"log/slog"
	"reflect"

	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
)

// WithDeps fills target's dependency fields from the sources in d, by field
// type, and returns it: WithDeps(d, usecases.Op{}) is an Op wired and ready.
// A field whose type no source provides is left zero; the use case will fail
// loudly when it needs it.
func WithDeps[T any](d Deps, target T) T {
	v := reflect.ValueOf(&target).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		if src, ok := source(d, f.Type()); ok {
			f.Set(src)
		}
	}
	return target
}

// source resolves one dependency by its declared field type.
func source(d Deps, t reflect.Type) (reflect.Value, bool) {
	switch t {
	case reflect.TypeOf((*interfaces.ConnectionRepository)(nil)).Elem():
		return reflect.ValueOf(d.Connections), true
	case reflect.TypeOf((*interfaces.ChangeRepository)(nil)).Elem():
		return reflect.ValueOf(d.Changes), true
	case reflect.TypeOf((*interfaces.GrantRepository)(nil)).Elem():
		return reflect.ValueOf(d.Grants), true
	case reflect.TypeOf((*interfaces.UserRepository)(nil)).Elem():
		return reflect.ValueOf(d.Users), true
	case reflect.TypeOf((*modules.Registry)(nil)).Elem():
		return reflect.ValueOf(d.Modules), true
	case reflect.TypeOf((*slog.Logger)(nil)):
		return reflect.ValueOf(d.Log), true
	case reflect.TypeOf(config.AuthInfo{}):
		return reflect.ValueOf(d.Auth), true
	}
	return reflect.Value{}, false
}
