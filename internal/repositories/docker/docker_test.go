package docker

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Asserts the exact docker argv for type every op, over a fake runner.
func TestContainerOps(t *testing.T) {
	var calls []string
	out := ""
	errOut := ""
	c := &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if errOut != "" {
			return []byte(errOut), errors.New("docker: " + errOut)
		}
		return []byte(out), nil
	}}

	opts := ContainerOptions{
		Name:         "noxis-zeus-postgres",
		Image:        "postgres:16-alpine",
		Labels:       map[string]string{"noxis.desired": "postgres|16-alpine", "noxis.node": "zeus"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "noxis"},
		Ports:        map[string]string{"5432": "5432"},
		Network:      "noxis-zeus",
		NetworkAlias: "postgres",
	}
	want := "run -d --name noxis-zeus-postgres --label noxis.desired=postgres|16-alpine --label noxis.node=zeus -e POSTGRES_PASSWORD=noxis -p 5432 --network noxis-zeus --network-alias postgres postgres:16-alpine"
	if err := c.ContainerRun(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := calls[len(calls)-1]; got != want {
		t.Fatalf("run argv:\n got %s\nwant %s", got, want)
	}

	out = `[{"State":{"Running":true},"Config":{"Image":"postgres:16-alpine","Labels":{"noxis.desired":"postgres|16-alpine"}}}]`
	ct, err := c.ContainerInspect(context.Background(), "noxis-zeus-postgres")
	if err != nil || !ct.State.Running || ct.Config.Image != "postgres:16-alpine" {
		t.Fatalf("inspect: %+v, %v", ct, err)
	}
	errOut = "Error: No such object"
	if _, err := c.ContainerInspect(context.Background(), "gone"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing inspect: %v", err)
	}
	errOut = ""

	out = "noxis-zeus-postgres\nnoxis-zeus-redis"
	names, err := c.ContainerList(context.Background(), "noxis.node=zeus")
	if err != nil || len(names) != 2 || names[0] != "noxis-zeus-postgres" {
		t.Fatalf("list: %v, %v", names, err)
	}

	if err := c.ContainerRemove(context.Background(), "noxis-zeus-postgres"); err != nil {
		t.Fatal(err)
	}
	if got := calls[len(calls)-1]; got != "rm -f noxis-zeus-postgres" {
		t.Fatalf("rm argv: %s", got)
	}
}
