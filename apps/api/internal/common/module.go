// Package common provides shared runtime context helpers.
package common

import (
	"context"
	"errors"
	"fmt"
)

// ErrModuleLoaderUnavailable reports that no module registry was attached to the request context.
var ErrModuleLoaderUnavailable = errors.New("module loader unavailable")

// ModuleLoader resolves a typed target from the process module registry.
type ModuleLoader interface {
	// LoadModule fills target with a registered module selected by its pointer type and context.
	LoadModule(ctx context.Context, target any) error
}

type moduleLoaderKey struct{}
type moduleFamilyKey struct{}

// WithModuleLoader attaches the process module registry to a context.
func WithModuleLoader(ctx context.Context, loader ModuleLoader) context.Context {
	return context.WithValue(ctx, moduleLoaderKey{}, loader)
}

// WithModuleFamily attaches the route-selected engine family to ctx for typed engine lookup.
func WithModuleFamily(ctx context.Context, family string) context.Context {
	return context.WithValue(ctx, moduleFamilyKey{}, family)
}

// ModuleFamily returns the route-selected engine family, if one was attached.
func ModuleFamily(ctx context.Context) (string, bool) {
	family, ok := ctx.Value(moduleFamilyKey{}).(string)
	return family, ok && family != ""
}

// LoadModule resolves target through the module registry attached to ctx.
func LoadModule(ctx context.Context, target any) error {
	loader, ok := ctx.Value(moduleLoaderKey{}).(ModuleLoader)
	if !ok || loader == nil {
		return ErrModuleLoaderUnavailable
	}
	if err := loader.LoadModule(ctx, target); err != nil {
		return fmt.Errorf("load module: %w", err)
	}
	return nil
}
