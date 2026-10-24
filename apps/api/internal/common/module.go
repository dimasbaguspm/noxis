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
	// LoadModule fills target with a registered module selected by its pointer type.
	LoadModule(target any) error
}

type moduleLoaderKey struct{}

// WithModuleLoader attaches the process module registry to a context.
func WithModuleLoader(ctx context.Context, loader ModuleLoader) context.Context {
	return context.WithValue(ctx, moduleLoaderKey{}, loader)
}

// LoadModule resolves target through the module registry attached to ctx.
func LoadModule(ctx context.Context, target any) error {
	loader, ok := ctx.Value(moduleLoaderKey{}).(ModuleLoader)
	if !ok || loader == nil {
		return ErrModuleLoaderUnavailable
	}
	if err := loader.LoadModule(target); err != nil {
		return fmt.Errorf("load module: %w", err)
	}
	return nil
}
