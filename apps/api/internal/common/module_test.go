package common

import (
	"context"
	"errors"
	"testing"
)

type moduleLoaderFunc func(any) error

func (f moduleLoaderFunc) LoadModule(target any) error { return f(target) }

func TestLoadModuleRequiresLoader(t *testing.T) {
	if err := LoadModule(context.Background(), new(string)); !errors.Is(err, ErrModuleLoaderUnavailable) {
		t.Fatalf("LoadModule error = %v, want %v", err, ErrModuleLoaderUnavailable)
	}
}

func TestLoadModuleDelegatesAndWrapsErrors(t *testing.T) {
	want := errors.New("not registered")
	ctx := WithModuleLoader(context.Background(), moduleLoaderFunc(func(target any) error {
		if _, ok := target.(*string); !ok {
			t.Fatalf("target type = %T, want *string", target)
		}
		return want
	}))
	if err := LoadModule(ctx, new(string)); !errors.Is(err, want) {
		t.Fatalf("LoadModule error = %v, want wrapped %v", err, want)
	}
}
