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
type moduleDriverKey struct{}

// WithModuleLoader attaches the process module registry to a context.
func WithModuleLoader(ctx context.Context, loader ModuleLoader) context.Context {
	return context.WithValue(ctx, moduleLoaderKey{}, loader)
}

// WithModuleDriver attaches the route-selected driver to ctx for typed module lookup.
func WithModuleDriver(ctx context.Context, driver string) context.Context {
	return context.WithValue(ctx, moduleDriverKey{}, driver)
}

// ModuleDriver returns the route-selected driver, if one was attached.
func ModuleDriver(ctx context.Context) (string, bool) {
	driver, ok := ctx.Value(moduleDriverKey{}).(string)
	return driver, ok && driver != ""
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
