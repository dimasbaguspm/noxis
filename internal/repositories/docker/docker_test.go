package docker

import (
	"context"
	"strings"
	"testing"
)

// Asserts the exact docker argv for every network op, over a fake runner.
func TestNetworkOps(t *testing.T) {
	var calls []string
	out := ""
	c := &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		return []byte(out), nil
	}}

	if err := c.NetworkCreate(context.Background(), "noxis-zeus", "noxis.network=noxis-zeus"); err != nil {
		t.Fatal(err)
	}
	if got := calls[len(calls)-1]; got != "network create --driver bridge --label noxis.network=noxis-zeus noxis-zeus" {
		t.Fatalf("create argv: %s", got)
	}

	out = "noxis-zeus\n"
	ok, err := c.NetworkExists(context.Background(), "noxis.network=noxis-zeus")
	if err != nil || !ok {
		t.Fatalf("exists: %v, %v", ok, err)
	}
	out = ""
	if ok, err := c.NetworkExists(context.Background(), "noxis.network=gone"); err != nil || ok {
		t.Fatalf("missing: %v, %v", ok, err)
	}

	if err := c.NetworkRemove(context.Background(), "noxis-zeus"); err != nil {
		t.Fatal(err)
	}
	if got := calls[len(calls)-1]; got != "network rm noxis-zeus" {
		t.Fatalf("rm argv: %s", got)
	}
}
