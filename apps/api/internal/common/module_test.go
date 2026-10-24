package common

import (
	"context"
	"errors"
	"testing"
)

type moduleLoaderFunc func(context.Context, any) error

func (f moduleLoaderFunc) LoadModule(ctx context.Context, target any) error { return f(ctx, target) }

func TestLoadModuleRequiresLoader(t *testing.T) {
	if err := LoadModule(context.Background(), new(string)); !errors.Is(err, ErrModuleLoaderUnavailable) {
		t.Fatalf("LoadModule error = %v, want %v", err, ErrModuleLoaderUnavailable)
	}
}

func TestLoadModuleDelegatesAndWrapsErrors(t *testing.T) {
	want := errors.New("not registered")
	ctx := WithModuleFamily(context.Background(), "sql")
	ctx = WithModuleLoader(ctx, moduleLoaderFunc(func(ctx context.Context, target any) error {
		if family, ok := ModuleFamily(ctx); !ok || family != "sql" {
			t.Fatalf("module family = %q, %t; want sql", family, ok)
		}
		if _, ok := target.(*string); !ok {
			t.Fatalf("target type = %T, want *string", target)
		}
		return want
	}))
	if err := LoadModule(ctx, new(string)); !errors.Is(err, want) {
		t.Fatalf("LoadModule error = %v, want wrapped %v", err, want)
	}
}
