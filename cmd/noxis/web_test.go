package main

import (
	"bytes"
	"strings"
	"testing"
)

// The console may be served from a host the API has never heard of, so the address it calls cannot
// be a default: a page that asks the reader's own machine for the API is a page that fails on every
// deployment but the developer's.
func TestWebServeNeedsTheAPI(t *testing.T) {
	cmd := newWeb()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("web serve started without an API")
	}
	if !strings.Contains(err.Error(), "--api") {
		t.Fatalf("error = %q, want it to name --api", err)
	}
}
