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

	// lifecycle verbs
	table := []struct{ verb, arg, want string }{
		{"ContainerStart", "noxis-zeus-postgres", "start noxis-zeus-postgres"},
		{"ContainerStop", "noxis-zeus-postgres", "stop noxis-zeus-postgres"},
		{"ContainerRestart", "noxis-zeus-postgres", "restart noxis-zeus-postgres"},
		{"ContainerKill", "noxis-zeus-postgres", "kill noxis-zeus-postgres"},
		{"ImagePull", "postgres:16-alpine", "pull postgres:16-alpine"},
		{"ImageRemove", "postgres:16-alpine", "image rm postgres:16-alpine"},
		{"VolumeCreate", "noxis-zeus-pgdata", "volume create noxis-zeus-pgdata"},
		{"VolumeRemove", "noxis-zeus-pgdata", "volume rm noxis-zeus-pgdata"},
	}
	var cerr error
	for _, tt := range table {
		switch tt.verb {
		case "ContainerStart":
			cerr = c.ContainerStart(context.Background(), tt.arg)
		case "ContainerStop":
			cerr = c.ContainerStop(context.Background(), tt.arg)
		case "ContainerRestart":
			cerr = c.ContainerRestart(context.Background(), tt.arg)
		case "ContainerKill":
			cerr = c.ContainerKill(context.Background(), tt.arg)
		case "ImagePull":
			cerr = c.ImagePull(context.Background(), tt.arg)
		case "ImageRemove":
			cerr = c.ImageRemove(context.Background(), tt.arg)
		case "VolumeCreate":
			cerr = c.VolumeCreate(context.Background(), tt.arg)
		case "VolumeRemove":
			cerr = c.VolumeRemove(context.Background(), tt.arg)
		}
		if cerr != nil {
			t.Fatalf("%s: %v", tt.verb, cerr)
		}
		if got := calls[len(calls)-1]; got != tt.want {
			t.Fatalf("%s argv: %s", tt.verb, got)
		}
	}

	// status / exists probes
	calls = calls[:0]
	out = "running"
	if st, err := c.ContainerStatus(context.Background(), "noxis-zeus-postgres"); err != nil || st != "running" {
		t.Fatalf("status: %q, %v", st, err)
	}
	if got := calls[len(calls)-1]; got != "inspect --format {{.State.Status}} noxis-zeus-postgres" {
		t.Fatalf("status argv: %s", got)
	}
	errOut = "Error: No such image"
	if ok, err := c.ImageExists(context.Background(), "ghost:1"); err != nil || ok {
		t.Fatalf("image missing: %v, %v", ok, err)
	}
	errOut = ""
	if ok, err := c.ImageExists(context.Background(), "postgres:16-alpine"); err != nil || !ok {
		t.Fatalf("image present: %v, %v", ok, err)
	}
	out = "noxis-zeus-pgdata"
	if ok, err := c.VolumeExists(context.Background(), "noxis-zeus-pgdata"); err != nil || !ok {
		t.Fatalf("volume exists: %v, %v", ok, err)
	}
	out = ""
	if ok, err := c.VolumeExists(context.Background(), "ghostvol"); err != nil || ok {
		t.Fatalf("volume missing: %v, %v", ok, err)
	}
}
