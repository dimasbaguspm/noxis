package docker

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ops returns a client backed by a fake runner that records argv; out/errOut
// are injected per-phase.
func ops() (c *Client, calls *[]string, setOut func(string), setErr func(string)) {
	var callList []string
	out := ""
	errOut := ""
	c = &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
		callList = append(callList, strings.Join(args, " "))
		if errOut != "" {
			return []byte(errOut), errors.New("docker: " + errOut)
		}
		return []byte(out), nil
	}}
	return c, &callList, func(s string) { out = s }, func(s string) { errOut = s }
}

// Asserts the exact docker argv for the network ops over a fake runner.
func TestNetworkOps(t *testing.T) {
	c, calls, setOut, _ := ops()
	setOut("")

	if err := c.NetworkCreate(context.Background(), "noxis-zeus", "noxis.network=noxis-zeus"); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[len(*calls)-1]; got != "network create --driver bridge --label noxis.network=noxis-zeus noxis-zeus" {
		t.Fatalf("create argv: %s", got)
	}

	setOut("noxis-zeus\n")
	ok, err := c.NetworkExists(context.Background(), "noxis.network=noxis-zeus")
	if err != nil || !ok {
		t.Fatalf("exists: %v, %v", ok, err)
	}
	setOut("")
	if ok, err := c.NetworkExists(context.Background(), "noxis.network=gone"); err != nil || ok {
		t.Fatalf("missing: %v, %v", ok, err)
	}

	if err := c.NetworkRemove(context.Background(), "noxis-zeus"); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[len(*calls)-1]; got != "network rm noxis-zeus" {
		t.Fatalf("rm argv: %s", got)
	}
}

// NetworkReconcile decides create/keep/remove from the label probe alone.
func TestNetworkReconcile(t *testing.T) {
	c, calls, setOut, _ := ops()
	setOut("")

	d, err := c.NetworkReconcile(context.Background(), "noxis-zeus", "noxis.network=noxis-zeus", 1)
	if err != nil || d != "network noxis-zeus created (1 engine(s))" {
		t.Fatalf("create: %q, %v", d, err)
	}
	if last := (*calls)[len(*calls)-1]; last != "network create --driver bridge --label noxis.network=noxis-zeus noxis-zeus" {
		t.Fatalf("create argv: %s", last)
	}

	setOut("noxis-zeus\n")
	before := len(*calls)
	if d, err := c.NetworkReconcile(context.Background(), "noxis-zeus", "noxis.network=noxis-zeus", 2); err != nil || d != "in sync" {
		t.Fatalf("in sync: %q, %v", d, err)
	}
	if len(*calls) != before+1 { // only the ls
		t.Fatalf("in-sync should only probe, calls=%d", len(*calls))
	}

	if d, err := c.NetworkReconcile(context.Background(), "noxis-zeus", "noxis.network=noxis-zeus", 0); err != nil || d != "network noxis-zeus removed (spec empty)" {
		t.Fatalf("rm: %q, %v", d, err)
	}
	if last := (*calls)[len(*calls)-1]; last != "network rm noxis-zeus" {
		t.Fatalf("rm argv: %s", last)
	}

	setOut("")
	before = len(*calls)
	if d, err := c.NetworkReconcile(context.Background(), "noxis-zeus", "noxis.network=noxis-zeus", 0); err != nil || d != "" {
		t.Fatalf("no-op: %q, %v", d, err)
	}
	if len(*calls) != before+1 {
		t.Fatalf("no-op should only probe, calls=%d", len(*calls))
	}
}

// Asserts the container argv, inspect decode, exec and status probes.
func TestContainerOps(t *testing.T) {
	c, calls, setOut, setErr := ops()

	opts := ContainerOptions{
		Name:         "noxis-zeus-postgres",
		Image:        "postgres:16-alpine",
		Labels:       map[string]string{"noxis.desired": "postgres|16-alpine", "noxis.node": "zeus"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "noxis"},
		Ports:        map[string]string{"5432": "5432"},
		Volumes:      []string{"noxis-zeus-postgres-16-data:/var/lib/postgresql/data"},
		Network:      "noxis-zeus",
		NetworkAlias: "postgres",
	}
	want := "run -d --name noxis-zeus-postgres --label noxis.desired=postgres|16-alpine --label noxis.node=zeus -e POSTGRES_PASSWORD=noxis -p 5432 -v noxis-zeus-postgres-16-data:/var/lib/postgresql/data --network noxis-zeus --network-alias postgres postgres:16-alpine"
	if err := c.ContainerRun(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[len(*calls)-1]; got != want {
		t.Fatalf("run argv:\n got %s\nwant %s", got, want)
	}

	setOut(`[{"State":{"Running":true},"Config":{"Image":"postgres:16-alpine","Labels":{"noxis.desired":"postgres|16-alpine"}}}]`)
	ct, err := c.ContainerInspect(context.Background(), "noxis-zeus-postgres")
	if err != nil || !ct.State.Running || ct.Config.Image != "postgres:16-alpine" {
		t.Fatalf("inspect: %+v, %v", ct, err)
	}
	setErr("Error: No such object")
	if _, err := c.ContainerInspect(context.Background(), "gone"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing inspect: %v", err)
	}
	setErr("")

	setOut("noxis-zeus-postgres\nnoxis-zeus-redis")
	names, err := c.ContainerList(context.Background(), "noxis.node=zeus")
	if err != nil || len(names) != 2 || names[0] != "noxis-zeus-postgres" {
		t.Fatalf("list: %v, %v", names, err)
	}

	// lifecycle verbs
	var cerr error
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
		if got := (*calls)[len(*calls)-1]; got != tt.want {
			t.Fatalf("%s argv: %s", tt.verb, got)
		}
	}

	// exec
	*calls = (*calls)[:0]
	setOut("1\n")
	res, err := c.ContainerExec(context.Background(), "noxis-zeus-postgres", "psql", "-U", "postgres", "-c", "SELECT 1")
	if err != nil || res != "1\n" {
		t.Fatalf("exec: %q, %v", res, err)
	}
	if got := (*calls)[len(*calls)-1]; got != "exec noxis-zeus-postgres psql -U postgres -c SELECT 1" {
		t.Fatalf("exec argv: %s", got)
	}
	setErr("Error: No such container")
	if _, err := c.ContainerExec(context.Background(), "gone", "true"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("exec missing: %v", err)
	}
	setErr("")

	// status / exists probes
	*calls = (*calls)[:0]
	setOut("running")
	if st, err := c.ContainerStatus(context.Background(), "noxis-zeus-postgres"); err != nil || st != "running" {
		t.Fatalf("status: %q, %v", st, err)
	}
	if got := (*calls)[len(*calls)-1]; got != "inspect --format {{.State.Status}} noxis-zeus-postgres" {
		t.Fatalf("status argv: %s", got)
	}
	setErr("Error: No such image")
	if ok, err := c.ImageExists(context.Background(), "ghost:1"); err != nil || ok {
		t.Fatalf("image missing: %v, %v", ok, err)
	}
	setErr("")
	if ok, err := c.ImageExists(context.Background(), "postgres:16-alpine"); err != nil || !ok {
		t.Fatalf("image present: %v, %v", ok, err)
	}
	setOut("noxis-zeus-pgdata")
	if ok, err := c.VolumeExists(context.Background(), "noxis-zeus-pgdata"); err != nil || !ok {
		t.Fatalf("volume exists: %v, %v", ok, err)
	}
	setOut("")
	if ok, err := c.VolumeExists(context.Background(), "ghostvol"); err != nil || ok {
		t.Fatalf("volume missing: %v, %v", ok, err)
	}

	if err := c.ContainerRemove(context.Background(), "noxis-zeus-postgres"); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[len(*calls)-1]; got != "rm -f noxis-zeus-postgres" {
		t.Fatalf("rm argv: %s", got)
	}
}
